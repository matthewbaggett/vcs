package github

import (
	"context"
	"io"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// The artifact verbs have no GitHub call yet. Each returns forge.ErrUnsupported, so the
// command says so and exits 69.

// CIArtifacts lists the artifacts of one run.
func (c *Client) CIArtifacts(context.Context, int64) ([]forge.Artifact, error) {
	return nil, forge.ErrUnsupported
}

// CIArtifactGet downloads one artifact.
func (c *Client) CIArtifactGet(context.Context, forge.Artifact, io.Writer) (int64, error) {
	return 0, forge.ErrUnsupported
}
