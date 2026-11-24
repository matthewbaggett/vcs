package forge

import "context"

// Protection is one branch protection rule.
type Protection struct {
	Rule      string
	Contexts  []string
	Approvals int
	Push      string // none, whitelist, or all
	Admins    bool
}

// ProtectionLister reads the branch protection rules of the repository.
type ProtectionLister interface {
	ProtectionList(ctx context.Context) ([]Protection, error)
}
