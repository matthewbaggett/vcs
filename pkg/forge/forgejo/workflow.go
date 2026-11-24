package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

// WorkflowDispatch starts one run of a workflow file at a branch or tag. The server answers 204
// with no body, so there is no run id to return.
func (c *Client) WorkflowDispatch(ctx context.Context, file, ref string) error {
	_, err := c.sendJSON(ctx, "POST", fmt.Sprintf("repos/%s/actions/workflows/%s/dispatches", c.cfg.Repo, url.PathEscape(file)), map[string]string{"ref": ref})
	return err
}
