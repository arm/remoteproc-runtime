package remoteproc

import "fmt"

type State string

const (
	StateOffline   State = "offline"
	StateSuspended State = "suspended"
	StateRunning   State = "running"
	StateCrashed   State = "crashed"
	StateInvalid   State = "invalid"
)

func NewState(value string) (State, error) {
	switch State(value) {
	case StateOffline:
		return StateOffline, nil
	case StateSuspended:
		return StateSuspended, nil
	case StateRunning:
		return StateRunning, nil
	case StateCrashed:
		return StateCrashed, nil
	case StateInvalid:
		return StateInvalid, nil
	default:
		return "", fmt.Errorf("unknown state %s", value)
	}
}
