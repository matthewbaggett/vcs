package forge

import "context"

// Runner is one CI runner of the whole instance.
type Runner struct {
	ID      int64
	Name    string
	Status  string
	Labels  []string
	UUID    string
	Version string
}

// RunnerLister is the optional interface of a driver that lists CI runners.
// The command asserts it, so a driver without it needs no stub.
type RunnerLister interface {
	// RunnerList reads the runners of the whole instance, sorted by id. The call needs an admin
	// token. A server that refuses the token gives ErrNoSession.
	RunnerList(ctx context.Context) ([]Runner, error)
}
