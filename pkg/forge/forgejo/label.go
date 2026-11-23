package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.LabelManager = (*Client)(nil)

// sendJSON sends one JSON body to the API and returns the answer.
func (c *Client) sendJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	return c.send(req, path)
}

// LabelList reads every label of the repository.
func (c *Client) LabelList(ctx context.Context) ([]forge.Label, error) {
	var out []forge.Label
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/labels?limit=50&page=%d", c.cfg.Repo, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Color       string `json:"color"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of labels")
		}
		for _, r := range rows {
			out = append(out, forge.Label{ID: r.ID, Name: r.Name, Color: strings.TrimPrefix(r.Color, "#"), Description: r.Description})
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}

// LabelCreate makes one label. The color is six hex digits, with or without a leading #.
func (c *Client) LabelCreate(ctx context.Context, name, color, description string) error {
	_, err := c.sendJSON(ctx, http.MethodPost, fmt.Sprintf("repos/%s/labels", c.cfg.Repo), map[string]string{
		"name": name, "color": "#" + strings.TrimPrefix(color, "#"), "description": description,
	})
	return err
}

// LabelRename changes the name of one label, found by id.
func (c *Client) LabelRename(ctx context.Context, id int64, name string) error {
	_, err := c.sendJSON(ctx, http.MethodPatch, fmt.Sprintf("repos/%s/labels/%d", c.cfg.Repo, id), map[string]string{"name": name})
	return err
}

var _ forge.BranchRenamer = (*Client)(nil)

// BranchRename renames a branch on the server. Each slash of the old name is encoded in the path.
func (c *Client) BranchRename(ctx context.Context, old, name string) error {
	path := fmt.Sprintf("repos/%s/branches/%s", c.cfg.Repo, strings.ReplaceAll(old, "/", "%2F"))
	_, err := c.sendJSON(ctx, http.MethodPatch, path, map[string]string{"name": name})
	return err
}
