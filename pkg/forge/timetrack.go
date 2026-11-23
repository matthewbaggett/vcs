package forge

import "context"

// TimeEntry is one logged span of time on an issue.
type TimeEntry struct {
	ID      int64
	User    string
	Seconds int64
	Created string
	Issue   int // the issue number, or 0
}

// Stopwatch is a stopwatch that runs for the logged-in user.
type Stopwatch struct {
	Repo    string // owner/name
	Issue   int
	Seconds int64
	Created string
}

// TimeTracker is the optional interface of a driver that tracks time.
// The command asserts it, so a driver without it needs no stub.
type TimeTracker interface {
	// TimeAdd logs seconds on an issue. It returns the new total of the issue, or -1 when the
	// write worked and the total could not be read.
	TimeAdd(ctx context.Context, n int, seconds int64) (int64, error)
	// TimeList reads the entries of one issue, or of the whole repository when n is 0.
	// With mine, it keeps only the entries of the logged-in user.
	TimeList(ctx context.Context, n int, mine bool) ([]TimeEntry, error)
	// StopwatchList reads the stopwatches of the logged-in user. The call covers the whole instance.
	StopwatchList(ctx context.Context) ([]Stopwatch, error)
}
