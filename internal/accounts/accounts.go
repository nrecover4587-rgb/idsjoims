// REPLACE existing file at: internal/accounts/accounts.go

package accounts

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/joinids/bot/internal/config"
	"github.com/joinids/bot/internal/database"
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

func newClient(sessionFile string) (*telegram.Client, error) {
	client, err := telegram.NewClient(telegram.ClientConfig{
		SessionFile: sessionFile,
		AppID:       config.C.APIID,
		AppHash:     config.C.APIHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mtproto client: %w", err)
	}
	return client, nil
}

func sessionFilePath(id string) string {
	return fmt.Sprintf("/tmp/session_%s.json", id)
}

func ValidatePyrogramSession(session string, addedBy int64) (database.Account, error) {
	sessionPath := sessionFilePath(fmt.Sprintf("pyrogram_%d", addedBy))

	client, err := newClient(sessionPath)
	if err != nil {
		return database.Account{}, err
	}

	me, err := client.UsersGetFullUser(&telegram.UsersGetFullUserParams{
		Id: &telegram.InputUserSelf{},
	})
	if err != nil {
		return database.Account{}, fmt.Errorf("session invalid or expired: %w", err)
	}

	var userID int32
	var phone string

	if me.FullUser != nil {
		userID = me.FullUser.ID
	}

	for _, u := range me.Users {
		if obj, ok := u.(*telegram.UserObj); ok {
			phone = obj.Phone
			userID = obj.ID
			break
		}
	}

	if phone == "" {
		phone = fmt.Sprintf("uid_%d", userID)
	}

	return database.Account{
		Type:        "pyrogram",
		SessionStr:  session,
		Phone:       phone,
		TelegramUID: int64(userID),
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
	sessionPath := sessionFilePath(fmt.Sprintf("direct_%d", userID))
	client, err := newClient(sessionPath)
	if err != nil {
		return err
	}

	result, err := client.AuthSendCode(&telegram.AuthSendCodeParams{
		PhoneNumber: phone,
		ApiId:       config.C.APIID,
		ApiHash:     config.C.APIHash,
		Settings:    &telegram.CodeSettings{},
	})
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

	code = strings.ReplaceAll(code, " ", "")

	authResult, err := session.Client.AuthSignIn(&telegram.AuthSignInParams{
		PhoneNumber:   session.Phone,
		PhoneCodeHash: session.SentHash,
		PhoneCode:     code,
	})
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

	acc, err := buildAccountFromAuth(authResult, session.Client, userID)
	return acc, false, err
}

func Submit2FA(userID int64, password string) (database.Account, error) {
	pendingLoginsMu.Lock()
	session, ok := pendingLogins[userID]
	pendingLoginsMu.Unlock()

	if !ok || !session.Phone2FA {
		return database.Account{}, fmt.Errorf("no pending 2FA session")
	}

	srpInfo, err := session.Client.AccountGetPassword()
	if err != nil {
		return database.Account{}, fmt.Errorf("failed to get 2FA info: %w", err)
	}

	srpAnswer, err := telegram.GetInputCheckPassword(password, srpInfo)
	if err != nil {
		return database.Account{}, fmt.Errorf("failed to compute SRP: %w", err)
	}

	authResult, err := session.Client.AuthCheckPassword(&telegram.AuthCheckPasswordParams{
		Password: srpAnswer,
	})
	if err != nil {
		return database.Account{}, fmt.Errorf("wrong 2FA password: %w", err)
	}

	return buildAccountFromAuth(authResult, session.Client, userID)
}

func buildAccountFromAuth(auth telegram.AuthAuthorization, client *telegram.Client, addedBy int64) (database.Account, error) {
	authObj, ok := auth.(*telegram.AuthAuthorizationObj)
	if !ok {
		return database.Account{}, fmt.Errorf("unexpected auth response type")
	}

	user, ok := authObj.User.(*telegram.UserObj)
	if !ok {
		return database.Account{}, fmt.Errorf("unexpected user type in auth response")
	}

	pendingLoginsMu.Lock()
	delete(pendingLogins, addedBy)
	pendingLoginsMu.Unlock()

	phone := user.Phone
	if phone == "" {
		phone = fmt.Sprintf("uid_%d", user.ID)
	}

	return database.Account{
		Type:        "direct",
		SessionStr:  sessionFilePath(fmt.Sprintf("direct_%d", addedBy)),
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
