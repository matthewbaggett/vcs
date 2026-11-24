package forge

import "context"

// PRStateChanger is the optional interface of a driver that turns a pull request back into a
// draft, closes it with no merge, or ticks the rebase box of a Renovate pull request. The
// command asserts it, so a driver without it needs no stub.
type PRStateChanger interface {
	// PRDraft turns a pull request back into a draft. A draft stays a draft.
	PRDraft(ctx context.Context, n int) error
	// PRClose closes a pull request with no merge.
	PRClose(ctx context.Context, n int) error
	// PRRebase ticks the rebase box in the body of a Renovate pull request. It returns false when
	// the box was ticked already. It fails when the body has no rebase box.
	PRRebase(ctx context.Context, n int) (bool, error)
}
