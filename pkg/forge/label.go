package forge

import "context"

// Label is one label of a repository.
type Label struct {
	ID          int64
	Name        string
	Color       string // six hex digits, with no leading #
	Description string
}

// LabelManager is the optional interface of a driver that lists, makes, and renames labels.
// The command asserts it, so a driver without it needs no stub.
type LabelManager interface {
	LabelList(ctx context.Context) ([]Label, error)
	LabelCreate(ctx context.Context, name, color, description string) error
	LabelRename(ctx context.Context, id int64, name string) error
}
