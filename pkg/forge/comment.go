package forge

import "context"

// CommentRow is one comment with its id, as `comment list` shows it.
type CommentRow struct {
	ID      int64
	Author  string
	Created string
	Updated string
	Body    string
}

// CommentManager is the optional interface of a driver that lists, edits, and deletes comments.
// The command asserts it, so a driver without it needs no stub.
type CommentManager interface {
	// CommentList reads every comment of an issue or a pull request.
	CommentList(ctx context.Context, n int) ([]CommentRow, error)
	// CommentEdit replaces the text of one comment, found by id.
	CommentEdit(ctx context.Context, id int64, body string) error
	// CommentDelete removes one comment, found by id.
	CommentDelete(ctx context.Context, id int64) error
}
