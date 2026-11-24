package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.PRStateChanger = (*Client)(nil)

var (
	rebaseTicked = regexp.MustCompile(`(?m)^[ \t]*- \[[xX]\][ \t]*<!-- rebase-check -->`)
	rebaseBox    = regexp.MustCompile(`(?m)^([ \t]*- )\[ \]([ \t]*<!-- rebase-check -->)`)
)

// prFields reads the title and the body of a pull request.
func (c *Client) prFields(ctx context.Context, n int) (title, body string, err error) {
	raw, err := c.call(ctx, http.MethodGet, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n), nil)
	if err != nil {
		return "", "", err
	}
	var pr struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal(raw, &pr); err != nil {
		return "", "", fmt.Errorf("the server did not return pull request %d", n)
	}
	return pr.Title, pr.Body, nil
}

// PRDraft puts the WIP: prefix on the title, which makes the pull request a draft.
func (c *Client) PRDraft(ctx context.Context, n int) error {
	title, _, err := c.prFields(ctx, n)
	if err != nil {
		return err
	}
	up := strings.ToUpper(strings.TrimSpace(title))
	if strings.HasPrefix(up, "WIP:") || strings.HasPrefix(up, "[WIP]") {
		return nil // already a draft
	}
	_, err = c.call(ctx, http.MethodPatch, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n), map[string]string{"title": "WIP: " + strings.TrimSpace(title)})
	return err
}

// PRClose closes a pull request with no merge.
func (c *Client) PRClose(ctx context.Context, n int) error {
	_, err := c.call(ctx, http.MethodPatch, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n), map[string]string{"state": "closed"})
	return err
}

// PRRebase ticks the rebase box of a Renovate pull request.
func (c *Client) PRRebase(ctx context.Context, n int) (bool, error) {
	_, body, err := c.prFields(ctx, n)
	if err != nil {
		return false, err
	}
	if !strings.Contains(body, "<!-- rebase-check -->") {
		return false, fmt.Errorf("pull request %d has no rebase box. Only a Renovate pull request has one", n)
	}
	if rebaseTicked.MatchString(body) {
		return false, nil
	}
	_, err = c.call(ctx, http.MethodPatch, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n), map[string]string{"body": rebaseBox.ReplaceAllString(body, "${1}[x]${2}")})
	return true, err
}
