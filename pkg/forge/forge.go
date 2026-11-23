// Package forge holds the one interface that every forge driver meets.
package forge

import (
	"context"
	"errors"
	"fmt"
)

// PRStatus is the state of one pull request.
type PRStatus struct {
	State           string // open, closed, or merged
	Draft           bool
	Mergeable       bool
	ConflictedFiles []string
	CI              string // failure, pending, warning, success, or none
	HeadSHA         string
	URL             string
}

// Run is one CI run.
type Run struct {
	ID     int64
	Index  int64
	Status string
	Ref    string
	SHA    string
	Title  string
	URL    string
}

// Job is one job of a CI run.
type Job struct {
	ID      int64
	Name    string
	Status  string
	Attempt int
	NoLog   bool // a wrapper job that calls a reusable workflow has no log
}

// RunFilter narrows a list of runs. A Limit of 0 reads every run, up to MaxRuns.
type RunFilter struct {
	Branch string
	SHA    string
	Status string
	Limit  int
}

// MaxRuns is the most runs that one list call reads.
const MaxRuns = 1000

// Client is the one interface of every driver. A command never names the host.
type Client interface {
	PRStatus(ctx context.Context, n int) (PRStatus, error)
	CIRuns(ctx context.Context, f RunFilter) ([]Run, error)
	CIJobs(ctx context.Context, runID int64) ([]Job, error)
	CILog(ctx context.Context, jobID int64) (string, error)
	// CIRerun restarts a run. With failedOnly it restarts the failed jobs only.
	// It needs a web session and returns ErrNoSession without one.
	CIRerun(ctx context.Context, runID int64, failedOnly bool) error
	// Attach uploads files to an issue or a pull request.
	Attach(ctx context.Context, n int, files []string) ([]Asset, error)
	// Comment posts one comment on an issue or a pull request.
	Comment(ctx context.Context, n int, body string) error
}

// Asset is one uploaded file.
type Asset struct {
	Name string
	URL  string
}

// ErrNotFound means the item does not exist.
var ErrNotFound = errors.New("not found")

// ErrNoLog means the job has no log file.
var ErrNoLog = errors.New("job has no log")

// ErrNoSession means the verb needs a web session and there is none.
var ErrNoSession = errors.New("no web session. Run: vcs session login, or vcs session import")

// ErrUnsupported means the host lacks the verb.
var ErrUnsupported = errors.New("unsupported on this host")

// Error is a failed call to the server.
type Error struct {
	Status int
	Path   string
	Body   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("server answered %d for %s", e.Status, e.Path)
}
