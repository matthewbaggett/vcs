package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.IssueEditor = (*Client)(nil)

// call sends one JSON request to the API and returns the body.
func (c *Client) call(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var rd *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+"/api/v1/"+path, rd)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.authorize(req)
	return c.send(req, path)
}

// idByName reads every page of a list route and returns the id of each item, by name.
func (c *Client) idByName(ctx context.Context, route, nameKey string) (map[string]int64, error) {
	out := map[string]int64{}
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/%s?state=all&limit=50&page=%d", c.cfg.Repo, route, page))
		if err != nil {
			return nil, err
		}
		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list for %s", route)
		}
		for _, r := range rows {
			name, _ := r[nameKey].(string)
			id, _ := r["id"].(float64)
			out[name] = int64(id)
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}

// IssueEdit changes the title, the body, the state, the milestone or the labels of an issue.
// It resolves each label and the milestone first, so an unknown name stops the call before
// anything changes.
func (c *Client) IssueEdit(ctx context.Context, n int, in forge.IssueEdit) error {
	patch := map[string]any{}
	if in.Title != nil {
		patch["title"] = *in.Title
	}
	if in.Body != nil {
		patch["body"] = *in.Body
	}
	if in.State != nil {
		patch["state"] = *in.State
	}
	if in.Milestone != nil {
		switch m := *in.Milestone; m {
		case "none":
			patch["milestone"] = 0
		default:
			ids, err := c.idByName(ctx, "milestones", "title")
			if err != nil {
				return err
			}
			if id, ok := ids[m]; ok {
				patch["milestone"] = id
			} else if num, err := strconv.ParseInt(m, 10, 64); err == nil {
				patch["milestone"] = num
			} else {
				return fmt.Errorf("milestone %q: %w", m, forge.ErrNotFound)
			}
		}
	}
	var add, rm []int64
	if len(in.AddLabels) > 0 || len(in.RmLabels) > 0 {
		ids, err := c.idByName(ctx, "labels", "name")
		if err != nil {
			return err
		}
		for _, l := range in.AddLabels {
			id, ok := ids[l]
			if !ok {
				return fmt.Errorf("label %q: %w", l, forge.ErrNotFound)
			}
			add = append(add, id)
		}
		for _, l := range in.RmLabels {
			id, ok := ids[l]
			if !ok {
				return fmt.Errorf("label %q: %w", l, forge.ErrNotFound)
			}
			rm = append(rm, id)
		}
	}
	base := fmt.Sprintf("repos/%s/issues/%d", c.cfg.Repo, n)
	if len(patch) > 0 {
		if _, err := c.call(ctx, http.MethodPatch, base, patch); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		if _, err := c.call(ctx, http.MethodPost, base+"/labels", map[string]any{"labels": add}); err != nil {
			return err
		}
	}
	for _, id := range rm {
		if _, err := c.call(ctx, http.MethodDelete, fmt.Sprintf("%s/labels/%d", base, id), nil); err != nil {
			return err
		}
	}
	return nil
}
