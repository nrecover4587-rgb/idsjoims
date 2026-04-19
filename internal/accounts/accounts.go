package accounts

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/joinids/bot/internal/config"
	"github.com/joinids/bot/internal/database"
	"github.com/xelaj/mtproto"
	"github.com/xelaj/mtproto/telegram"
)

type ClientEntry struct {
	Client  *telegram.Client
	Account database.Account
	mu      sync.Mutex
}

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*ClientEntry
}

var M = &Manager{
	clients: make(map[string]*ClientEntry),
}

type LoginSession struct {
	Phone     string
	Phone2FA  bool
	SentHash  string
	Client    *telegram.Client
	CreatedAt time.Time
}

var (
	pendingLogins   = make(map[int64]*LoginSession)
	pendingLoginsMu sync.Mutex
)

func newClient(sessionData string) (*telegram.Client, error) {
	cfg := &mtproto.Config{
		AppID:   int32(config.C.APIID),
		AppHash: config.C.APIHash,
	}

	if sessionData != "" {
		raw, err := base64.StdEncoding.DecodeString(sessionData)
		if err == nil {
			cfg.Session = raw
		}
	}

	app, err := telegram.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create mtproto client: %w", err)
	}
	return app, nil
}

func ValidatePyrogramSession(session string, addedBy int64) (database.Account, error) {
	client, err := newClient(session)
	if err != nil {
		return database.Account{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	me, err := client.UsersGetFullUser(ctx, &telegram.InputUserSelf{})
	if err != nil {
		return database.Account{}, fmt.Errorf("session invalid or expired: %w", err)
	}

	user := me.Users[0].(*telegram.UserObj)
	phone := user.Phone
	if phone == "" {
		phone = fmt.Sprintf("uid_%d", user.ID)
	}

	return database.Account{
		Type:        "pyrogram",
		SessionStr:  session,
		Phone:       phone,
		TelegramUID: int64(user.ID),
		AddedBy:     addedBy,
	}, nil
}

func ValidateTelethonSession(session string, addedBy int64) (database.Account, error) {
	decoded, err := base64.URLEncoding.DecodeString(session)
	if err != nil {
		return database.Account{}, fmt.Errorf("invalid telethon session format")
	}

	b64Standard := base64.StdEncoding.EncodeToString(decoded)
	return ValidatePyrogramSession(b64Standard, addedBy)
}

func StartDirectLogin(userID int64, phone string) error {
	client, err := newClient("")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := client.AuthSendCode(ctx, phone, int32(config.C.APIID), config.C.APIHash, &telegram.CodeSettings{})
	if err != nil {
		return fmt.Errorf("failed to send OTP: %w", err)
	}

	pendingLoginsMu.Lock()
	pendingLogins[userID] = &LoginSession{
		Phone:     phone,
		SentHash:  result.PhoneCodeHash,
		Client:    client,
		CreatedAt: time.Now(),
	}
	pendingLoginsMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		pendingLoginsMu.Lock()
		if s, ok := pendingLogins[userID]; ok && s.Phone == phone {
			delete(pendingLogins, userID)
		}
		pendingLoginsMu.Unlock()
	}()

	return nil
}

func SubmitCode(userID int64, code string) (database.Account, bool, error) {
	pendingLoginsMu.Lock()
	session, ok := pendingLogins[userID]
	pendingLoginsMu.Unlock()

	if !ok {
		return database.Account{}, false, fmt.Errorf("no pending login, start again")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	code = strings.ReplaceAll(code, " ", "")

	auth, err := session.Client.AuthSignIn(ctx, session.Phone, session.SentHash, code)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "SESSION_PASSWORD_NEEDED") {
			pendingLoginsMu.Lock()
			pendingLogins[userID].Phone2FA = true
			pendingLoginsMu.Unlock()
			return database.Account{}, true, nil
		}
		return database.Account{}, false, fmt.Errorf("invalid code: %w", err)
	}

	return buildAccountFromAuth(auth, session.Client, userID)
}

func Submit2FA(userID int64, password string) (database.Account, error) {
	pendingLoginsMu.Lock()
	session, ok := pendingLogins[userID]
	pendingLoginsMu.Unlock()

	if !ok || !session.Phone2FA {
		return database.Account{}, fmt.Errorf("no pending 2FA session")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	srpInfo, err := session.Client.AccountGetPassword(ctx)
	if err != nil {
		return database.Account{}, fmt.Errorf("failed to get 2FA info: %w", err)
	}

	srpAnswer, err := telegram.GetInputCheckPassword(password, srpInfo)
	if err != nil {
		return database.Account{}, fmt.Errorf("failed to compute SRP: %w", err)
	}

	auth, err := session.Client.AuthCheckPassword(ctx, srpAnswer)
	if err != nil {
		return database.Account{}, fmt.Errorf("wrong 2FA password: %w", err)
	}

	return buildAccountFromAuth(auth, session.Client, userID)
}

func buildAccountFromAuth(auth *telegram.AuthAuthorization, client *telegram.Client, addedBy int64) (database.Account, error) {
	user, ok := auth.User.(*telegram.UserObj)
	if !ok {
		return database.Account{}, fmt.Errorf("unexpected user type in auth response")
	}

	raw := client.ExportSession()
	sessionStr := base64.StdEncoding.EncodeToString(raw)

	pendingLoginsMu.Lock()
	delete(pendingLogins, addedBy)
	pendingLoginsMu.Unlock()

	phone := user.Phone
	if phone == "" {
		phone = fmt.Sprintf("uid_%d", user.ID)
	}

	return database.Account{
		Type:        "pyrogram",
		SessionStr:  sessionStr,
		Phone:       phone,
		TelegramUID: int64(user.ID),
		AddedBy:     addedBy,
	}, nil
}

func HasPendingLogin(userID int64) bool {
	pendingLoginsMu.Lock()
	defer pendingLoginsMu.Unlock()
	_, ok := pendingLogins[userID]
	return ok
}

func IsPending2FA(userID int64) bool {
	pendingLoginsMu.Lock()
	defer pendingLoginsMu.Unlock()
	s, ok := pendingLogins[userID]
	return ok && s.Phone2FA
}

func ClearPendingLogin(userID int64) {
	pendingLoginsMu.Lock()
	defer pendingLoginsMu.Unlock()
	delete(pendingLogins, userID)
}

func GetClient(acc database.Account) (*telegram.Client, error) {
	M.mu.RLock()
	entry, exists := M.clients[acc.Phone]
	M.mu.RUnlock()

	if exists {
		return entry.Client, nil
	}

	client, err := newClient(acc.SessionStr)
	if err != nil {
		return nil, err
	}

	M.mu.Lock()
	M.clients[acc.Phone] = &ClientEntry{Client: client, Account: acc}
	M.mu.Unlock()

	return client, nil
}

func ReleaseClient(phone string) {
	M.mu.Lock()
	delete(M.clients, phone)
	M.mu.Unlock()
}
