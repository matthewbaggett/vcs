// Package forge holds the one interface that every forge driver meets.
package forge

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// Check is the newest status of one CI context on a commit.
type Check struct {
	Context string
	Status  string // success, failure, error, pending, or warning
	URL     string
}

// WatchState is what a watch reads in one poll: the pull request state, the head commit, the
// base branch, the newest status of each context, and the contexts that the base branch requires.
// A required context is a glob. An unreadable rule leaves Required empty.
type WatchState struct {
	State    string // open, closed, or merged
	HeadSHA  string
	Base     string
	Checks   []Check
	Required []string
}

// PRWatcher is the optional interface of a driver that reads a WatchState. A driver without it
// gets a state that the command builds from PRStatus.
type PRWatcher interface {
	PRWatch(ctx context.Context, n int) (WatchState, error)
}

// CommitTimer is an optional call: the commit time of one commit, in UTC, such as
// 2026-10-17T10:00:00Z. A driver that has no such call leaves it out.
type CommitTimer interface {
	CommitTime(ctx context.Context, sha string) (string, error)
}

// CommitStatus is one status row that a CI check posted on a commit. A commit holds many rows for
// one context. The row with the highest ID is the newest.
type CommitStatus struct {
	ID          int64
	Context     string
	Status      string // success, pending, failure, error, or warning
	URL         string // a full web address, or empty
	Description string
}

// StatusLister is the optional interface of a driver that lists every status row of a commit.
type StatusLister interface {
	CommitStatuses(ctx context.Context, sha string) ([]CommitStatus, error)
}

// TimelineEvent is one event on the timeline of an issue or a pull request. Detail holds the
// text that the command prints for the type: a label change as +name or -name, a title change
// as old -> new, a push as push <sha>,<sha>, and for other types the text of the event.
type TimelineEvent struct {
	ID      int64
	Created string
	Type    string
	User    string
	Detail  string
}

// Timeliner is the optional interface of a driver that lists the timeline of one issue.
type Timeliner interface {
	IssueTimeline(ctx context.Context, n int) ([]TimelineEvent, error)
}

// NewItem is one row of a list of the issues newer than a mark. Kind is issue or pr.
type NewItem struct {
	Number int
	Kind   string
	Author string
	Title  string
}

// Newer is the optional interface of a driver that lists the items above a mark. Issues and pull
// requests share one index space. With anyKind a pull request is listed too. State is open or all.
// TopIndex returns the highest index of any issue or pull request, in any state, or 0.
type Newer interface {
	IssuesNewer(ctx context.Context, mark int, state string, anyKind bool) ([]NewItem, error)
	TopIndex(ctx context.Context) (int, error)
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

// Artifact is one file bundle that a CI run uploaded. The server ties it to the run, not to a job.
type Artifact struct {
	ID      int64
	Name    string
	Size    int64
	Expired bool
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
	// CIArtifacts lists the artifacts of one run.
	CIArtifacts(ctx context.Context, runID int64) ([]Artifact, error)
	// CIArtifactGet writes the zip archive of one artifact to w and returns the bytes written.
	CIArtifactGet(ctx context.Context, a Artifact, w io.Writer) (int64, error)
	// CIRerun restarts a run. With failedOnly it restarts the failed jobs only.
	// It needs a web session and returns ErrNoSession without one.
	CIRerun(ctx context.Context, runID int64, failedOnly bool) error
	// Attach uploads files to an issue or a pull request.
	Attach(ctx context.Context, n int, files []string) ([]Asset, error)
	// Comment posts one comment on an issue or a pull request. It returns the comment id and
	// the web address of the comment.
	Comment(ctx context.Context, n int, body string) (int64, string, error)
	// CreatePR opens a pull request and returns its number and web address.
	CreatePR(ctx context.Context, in NewPR) (int, string, error)
	// CreateIssue opens an issue and returns its number and web address.
	CreateIssue(ctx context.Context, title, body string) (int, string, error)
	// PRReady takes a pull request out of draft.
	PRReady(ctx context.Context, n int) error
	// BoardList lists the project boards. With closed it lists the closed ones.
	BoardList(ctx context.Context, closed bool) ([]Board, error)
	// BoardView lists the cards of one board, column by column.
	BoardView(ctx context.Context, board string) ([]Card, error)
	// BoardAdd puts issues or pull requests on a board. It needs a web session on Forgejo.
	BoardAdd(ctx context.Context, board string, numbers []int) error
	// BoardMove moves cards to a column, named by id or by title. A card that is not on
	// the board is added. It needs a web session on Forgejo.
	BoardMove(ctx context.Context, board, column string, numbers []int) error
}

// Board is one project board.
type Board struct {
	ID    string
	Title string
	URL   string
}

// Card is one card on a board.
type Card struct {
	ColumnID string
	Column   string
	ItemID   string // the id that the server routes take
	Number   int
	Title    string
}

// NewPR is the input of CreatePR.
type NewPR struct {
	Title string
	Body  string
	Head  string
	Base  string
	Draft bool
}

// PREdit is the input of PREditor.PREdit. A nil field stays as it is.
type PREdit struct {
	Title *string
	Body  *string
	Base  *string
}

// PREditor is the optional interface of a driver that can change the title, the body or the base
// branch of a pull request. The command asserts it, so a driver without it needs no stub.
type PREditor interface {
	PREdit(ctx context.Context, n int, in PREdit) error
}

// PRMerge is the input of PRMerger.PRMerge. An empty Method means merge.
type PRMerge struct {
	Method       string // merge, rebase, rebase-merge, or squash
	DeleteBranch bool
}

// PRMerger is the optional interface of a driver that merges a pull request. The command
// asserts it, so a driver without it needs no stub.
type PRMerger interface {
	PRMerge(ctx context.Context, n int, in PRMerge) error
}

// PRInfo is one row of a pull request list.
type PRInfo struct {
	Number    int
	Title     string
	Head      string
	Base      string
	Mergeable bool
	CI        string // set only when the list asked for it
	Labels    []string
	Milestone string
}

// IssueInfo is one issue. List rows leave Body and Comments empty.
type IssueInfo struct {
	Number    int
	Title     string
	State     string
	Labels    []string
	Milestone string
	Body      string
	Comments  []Comment
	ID        int64  // the global id of the issue, which differs from the number. Zero when the driver has none
	URL       string // the web address of the issue. Empty when the driver has none
}

// Comment is one comment on an issue or a pull request.
type Comment struct {
	Author  string
	Created string
	Body    string
}

// ListFilter narrows a list of pull requests or issues. The driver reads every page.
// An empty State means open. Labels keeps only items that carry every label.
type ListFilter struct {
	State     string // open, closed, or all
	Labels    []string
	Milestone string
	Query     string // a text search of the title and body. Only issue lists use it
	Limit     int    // keep the newest n after the filter. 0 keeps all
}

// Reader is the optional interface of a driver that lists pull requests and issues.
// The command asserts it, so a driver without it needs no stub.
type Reader interface {
	PRList(ctx context.Context, f ListFilter) ([]PRInfo, error)
	IssueList(ctx context.Context, f ListFilter) ([]IssueInfo, error)
	IssueView(ctx context.Context, n int) (IssueInfo, error)
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
