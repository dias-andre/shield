package core

import "time"

type VaultInitState int

const (
	InitNotStarted VaultInitState = iota
	InitLoading
	InitSuccess
	InitFailed
	InitDeferredLazy
)

func (vis VaultInitState) String() string {
	return [...]string{
		"not_started",
		"loading",
		"success",
		"failed",
		"deferred_lazy",
	}[vis]
}

type InitializationRecord struct {
	State       VaultInitState
	Timestamp   time.Time
	ErrorMsg    string
	DeferReason string
}
