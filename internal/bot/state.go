package bot

import "github.com/joinids/bot/internal/state"

type StateKey = state.StateKey
type UserState = state.UserState

const (
	StateNone           = state.StateNone
	StateAddPyrogram    = state.StateAddPyrogram
	StateAddTelethon    = state.StateAddTelethon
	StateAddDirectPhone = state.StateAddDirectPhone
	StateAddDirectCode  = state.StateAddDirectCode
	StateAddDirect2FA   = state.StateAddDirect2FA
	StateAddDBChannel   = state.StateAddDBChannel
	StateAddSudoer      = state.StateAddSudoer
	StateBroadcast      = state.StateBroadcast
	StateJoinManual     = state.StateJoinManual
	StateJoinRange      = state.StateJoinRange
)

var States = state.States
