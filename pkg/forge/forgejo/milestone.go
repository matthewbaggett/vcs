package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.MilestoneManager = (*Client)(nil)

type milestoneRow struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	State       string  `json:"state"`
	Open        int     `json:"open_issues"`
	Closed      int     `json:"closed_issues"`
	Due         *string `json:"due_on"`
	Description string  `json:"description"`
}

func (r milestoneRow) out() forge.Milestone {
	m := forge.Milestone{ID: r.ID, Title: r.Title, State: r.State, Open: r.Open, Closed: r.Closed, Description: r.Description}
	if r.Due != nil && len(*r.Due) >= 10 {
		m.Due = (*r.Due)[:10]
	}
	return m
}

// MilestoneList reads every milestone in a state.
func (c *Client) MilestoneList(ctx context.Context, state string) ([]forge.Milestone, error) {
	var out []forge.Milestone
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/milestones?state=%s&limit=50&page=%d", c.cfg.Repo, state, page))
		if err != nil {
			return nil, err
		}
		var rows []milestoneRow
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of milestones")
		}
		for _, r := range rows {
			out = append(out, r.out())
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}

// MilestoneView finds one milestone by id or by title.
func (c *Client) MilestoneView(ctx context.Context, ref string) (forge.Milestone, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/milestones/%d", c.cfg.Repo, id))
		if err != nil {
			return forge.Milestone{}, err
		}
		var r milestoneRow
		if err := json.Unmarshal(body, &r); err != nil {
			return forge.Milestone{}, fmt.Errorf("the server did not return a milestone")
		}
		return r.out(), nil
	}
	all, err := c.MilestoneList(ctx, "all")
	if err != nil {
		return forge.Milestone{}, err
	}
	for _, m := range all {
		if m.Title == ref {
			return m, nil
		}
	}
	return forge.Milestone{}, fmt.Errorf("milestone %q: %w", ref, forge.ErrNotFound)
}

// MilestoneCreate makes one milestone. It refuses a title that exists, open or closed.
func (c *Client) MilestoneCreate(ctx context.Context, title, due, description string) error {
	all, err := c.MilestoneList(ctx, "all")
	if err != nil {
		return err
	}
	for _, m := range all {
		if strings.EqualFold(m.Title, title) {
			return fmt.Errorf("a milestone named %q exists already", title)
		}
	}
	payload := map[string]string{"title": title, "description": description}
	if due != "" {
		payload["due_on"] = due + "T00:00:00Z"
	}
	raw, _ := json.Marshal(payload)
	path := fmt.Sprintf("repos/%s/milestones", c.cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	_, err = c.send(req, path)
	return err
}
