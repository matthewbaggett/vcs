package forge

import "context"

// Milestone is one milestone of a repository.
type Milestone struct {
	ID          int64
	Title       string
	State       string
	Open        int
	Closed      int
	Due         string // YYYY-MM-DD, or empty
	Description string
}

// MilestoneManager is the optional interface of a driver that lists, views, and makes milestones.
// The command asserts it, so a driver without it needs no stub.
type MilestoneManager interface {
	// MilestoneList reads the milestones in a state: open, closed, or all.
	MilestoneList(ctx context.Context, state string) ([]Milestone, error)
	// MilestoneView finds one milestone by numeric id or by title. A closed one counts too.
	MilestoneView(ctx context.Context, ref string) (Milestone, error)
	// MilestoneCreate makes one milestone. It fails when the title exists. Due is YYYY-MM-DD or empty.
	MilestoneCreate(ctx context.Context, title, due, description string) error
}
