package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.TimeTracker = (*Client)(nil)

// TimeAdd logs seconds on an issue. It makes one attempt, because a retry could log the time twice.
func (c *Client) TimeAdd(ctx context.Context, n int, seconds int64) (int64, error) {
	raw, _ := json.Marshal(map[string]int64{"time": seconds})
	path := fmt.Sprintf("repos/%s/issues/%d/times", c.cfg.Repo, n)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	body, err := c.send(req, path)
	if err != nil {
		return 0, err
	}
	var made struct {
		ID int64 `json:"id"`
	}
	if json.Unmarshal(body, &made) != nil || made.ID == 0 {
		return 0, fmt.Errorf("the server did not return the time entry")
	}
	// The write is done. A failed read of the total must not look like a failed write.
	entries, err := c.TimeList(ctx, n, false)
	if err != nil {
		return -1, nil
	}
	var total int64
	for _, e := range entries {
		total += e.Seconds
	}
	return total, nil
}

// TimeList reads the time entries of one issue, or of the whole repository when n is 0.
func (c *Client) TimeList(ctx context.Context, n int, mine bool) ([]forge.TimeEntry, error) {
	route := fmt.Sprintf("repos/%s/times", c.cfg.Repo)
	if n > 0 {
		route = fmt.Sprintf("repos/%s/issues/%d/times", c.cfg.Repo, n)
	}
	var out []forge.TimeEntry
	first := int64(-1)
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("%s?limit=50&page=%d", route, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			ID      int64  `json:"id"`
			User    string `json:"user_name"`
			Time    int64  `json:"time"`
			Created string `json:"created"`
			Issue   *struct {
				Number int `json:"number"`
			} `json:"issue"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of time entries")
		}
		if len(rows) == 0 || rows[0].ID == first {
			break // an empty page, or a server that ignores the page and repeats page one
		}
		if first == -1 {
			first = rows[0].ID
		}
		for _, r := range rows {
			e := forge.TimeEntry{ID: r.ID, User: r.User, Seconds: r.Time, Created: r.Created}
			if r.Issue != nil {
				e.Issue = r.Issue.Number
			}
			out = append(out, e)
		}
		if len(rows) < 50 {
			break
		}
	}
	if !mine {
		return out, nil
	}
	body, err := c.get(ctx, "user")
	if err != nil {
		return nil, err
	}
	var me struct {
		Login string `json:"login"`
	}
	if json.Unmarshal(body, &me) != nil || me.Login == "" {
		return nil, fmt.Errorf("the server did not name the logged-in user. mine needs a login")
	}
	kept := out[:0]
	for _, e := range out {
		if e.User == me.Login {
			kept = append(kept, e)
		}
	}
	return kept, nil
}

// StopwatchList reads the stopwatches that run for the logged-in user.
func (c *Client) StopwatchList(ctx context.Context) ([]forge.Stopwatch, error) {
	body, err := c.get(ctx, "user/stopwatches")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Created string `json:"created"`
		Index   int    `json:"issue_index"`
		Owner   string `json:"repo_owner_name"`
		Repo    string `json:"repo_name"`
		Seconds int64  `json:"seconds"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("the server did not return a list of stopwatches")
	}
	out := make([]forge.Stopwatch, 0, len(rows))
	for _, r := range rows {
		out = append(out, forge.Stopwatch{Repo: r.Owner + "/" + r.Repo, Issue: r.Index, Seconds: r.Seconds, Created: r.Created})
	}
	return out, nil
}
