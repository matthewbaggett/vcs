package github

import (
	"context"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// The verbs below have no GitHub call in this change. Each returns forge.ErrUnsupported, so
// the command says so and exits 69. The write change replaces this file.

// CIRerun restarts a run.
func (c *Client) CIRerun(context.Context, int64, bool) error { return forge.ErrUnsupported }

// Attach uploads files to an issue or a pull request.
func (c *Client) Attach(context.Context, int, []string) ([]forge.Asset, error) {
	return nil, forge.ErrUnsupported
}

// Comment posts one comment.
func (c *Client) Comment(context.Context, int, string) error { return forge.ErrUnsupported }
