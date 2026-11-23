package forge

import "context"

// IssueEdit is the input of IssueEditor.IssueEdit. A nil or empty field stays as it is.
type IssueEdit struct {
	Title     *string
	Body      *string
	State     *string // open or closed
	Milestone *string // a title or an id. "none" clears it
	AddLabels []string
	RmLabels  []string
}

// IssueEditor is the optional interface of a driver that changes an issue. The command
// asserts it, so a driver without it needs no stub.
type IssueEditor interface {
	IssueEdit(ctx context.Context, n int, in IssueEdit) error
}
