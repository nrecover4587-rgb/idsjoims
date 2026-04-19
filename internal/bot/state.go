package bot

import "sync"

type StateKey string

const (
	StateNone           StateKey = ""
	StateAddPyrogram    StateKey = "add_pyrogram"
	StateAddTelethon    StateKey = "add_telethon"
	StateAddDirectPhone StateKey = "add_direct_phone"
	StateAddDirectCode  StateKey = "add_direct_code"
	StateAddDirect2FA   StateKey = "add_direct_2fa"
	StateAddDBChannel   StateKey = "add_db_channel"
	StateAddSudoer      StateKey = "add_sudoer"
	StateBroadcast      StateKey = "broadcast"
	StateJoinManual     StateKey = "join_manual"
	StateJoinRange      StateKey = "join_range"
)

type UserState struct {
	Key        StateKey
	Data       map[string]interface{}
}

type StateStore struct {
	mu     sync.RWMutex
	states map[int64]*UserState
}

var States = &StateStore{
	states: make(map[int64]*UserState),
}

func (s *StateStore) Set(userID int64, key StateKey, data map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if data == nil {
		data = make(map[string]interface{})
	}
	s.states[userID] = &UserState{Key: key, Data: data}
}

func (s *StateStore) Get(userID int64) *UserState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.states[userID]
	if !ok {
		return &UserState{Key: StateNone, Data: make(map[string]interface{})}
	}
	return st
}

func (s *StateStore) Clear(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
}
