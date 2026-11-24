package github

import (
	"context"
	"fmt"
	"net/http"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.PRStateChanger = (*Client)(nil)

// PRDraft turns a pull request back into a draft. REST has no route for it, so this uses GraphQL.
func (c *Client) PRDraft(ctx context.Context, n int) error {
	var q struct {
		Repository struct {
			PullRequest struct {
				ID      string `json:"id"`
				IsDraft bool   `json:"isDraft"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}
	if err := c.gql(ctx, `query($owner:String!,$name:String!,$n:Int!){repository(owner:$owner,name:$name){pullRequest(number:$n){id isDraft}}}`,
		c.repoVars(map[string]any{"n": n}), &q); err != nil {
		return err
	}
	pr := q.Repository.PullRequest
	if pr.ID == "" {
		return fmt.Errorf("pull request %d: %w", n, forge.ErrNotFound)
	}
	if pr.IsDraft {
		return nil
	}
	return c.gql(ctx, `mutation($id:ID!){convertPullRequestToDraft(input:{pullRequestId:$id}){pullRequest{id}}}`,
		map[string]any{"id": pr.ID}, nil)
}

// PRClose closes a pull request with no merge.
func (c *Client) PRClose(ctx context.Context, n int) error {
	_, err := c.send(ctx, http.MethodPatch, fmt.Sprintf("%s/repos/%s/pulls/%d", c.cfg.BaseURL, c.cfg.Repo, n), map[string]string{"state": "closed"})
	return err
}

// PRRebase has no GitHub form. The rebase box is a Renovate habit of Forgejo.
func (c *Client) PRRebase(ctx context.Context, n int) (bool, error) {
	return false, forge.ErrUnsupported
}
