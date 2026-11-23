package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.Reader = (*Client)(nil)

type labelJSON struct {
	Name string `json:"name"`
}

type itemJSON struct {
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	State     string      `json:"state"`
	Body      string      `json:"body"`
	Mergeable bool        `json:"mergeable"`
	Labels    []labelJSON `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (it itemJSON) labels() []string {
	out := make([]string, 0, len(it.Labels))
	for _, l := range it.Labels {
		out = append(out, l.Name)
	}
	return out
}

func (it itemJSON) milestone() string {
	if it.Milestone == nil {
		return ""
	}
	return it.Milestone.Title
}

// keep applies the filter that the server can't apply the same way for each route.
func keep(it itemJSON, f forge.ListFilter) bool {
	if f.Milestone != "" && it.milestone() != f.Milestone {
		return false
	}
	have := map[string]bool{}
	for _, l := range it.labels() {
		have[l] = true
	}
	for _, l := range f.Labels {
		if !have[l] {
			return false
		}
	}
	return true
}

// listItems reads every page of a list route, newest first, and applies the filter.
func (c *Client) listItems(ctx context.Context, route string, q url.Values, f forge.ListFilter) ([]itemJSON, error) {
	state := f.State
	if state == "" {
		state = "open"
	}
	q.Set("state", state)
	q.Set("limit", "50")
	if f.Query != "" {
		q.Set("q", f.Query)
	}
	var out []itemJSON
	for page := 1; page <= 100; page++ {
		q.Set("page", fmt.Sprint(page))
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/%s?%s", c.cfg.Repo, route, q.Encode()))
		if err != nil {
			return nil, err
		}
		var rows []itemJSON
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list for %s", route)
		}
		for _, it := range rows {
			if keep(it, f) {
				out = append(out, it)
			}
		}
		if len(rows) < 50 {
			break
		}
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// PRList lists pull requests, newest first.
func (c *Client) PRList(ctx context.Context, f forge.ListFilter) ([]forge.PRInfo, error) {
	items, err := c.listItems(ctx, "pulls", url.Values{}, f)
	if err != nil {
		return nil, err
	}
	out := make([]forge.PRInfo, 0, len(items))
	for _, it := range items {
		out = append(out, forge.PRInfo{
			Number: it.Number, Title: it.Title, Head: it.Head.Ref, Base: it.Base.Ref,
			Mergeable: it.Mergeable, Labels: it.labels(), Milestone: it.milestone(),
		})
	}
	return out, nil
}

// IssueList lists issues, newest first. The issue route also lists pull requests, so the
// query asks for issues only.
func (c *Client) IssueList(ctx context.Context, f forge.ListFilter) ([]forge.IssueInfo, error) {
	items, err := c.listItems(ctx, "issues", url.Values{"type": {"issues"}}, f)
	if err != nil {
		return nil, err
	}
	out := make([]forge.IssueInfo, 0, len(items))
	for _, it := range items {
		out = append(out, forge.IssueInfo{
			Number: it.Number, Title: it.Title, State: it.State,
			Labels: it.labels(), Milestone: it.milestone(),
		})
	}
	return out, nil
}

// IssueView reads one issue with its body and its comments.
func (c *Client) IssueView(ctx context.Context, n int) (forge.IssueInfo, error) {
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/issues/%d", c.cfg.Repo, n))
	if err != nil {
		return forge.IssueInfo{}, err
	}
	var it itemJSON
	if err := json.Unmarshal(body, &it); err != nil || it.Number == 0 {
		return forge.IssueInfo{}, fmt.Errorf("the server did not return issue %d", n)
	}
	out := forge.IssueInfo{
		Number: it.Number, Title: it.Title, State: it.State,
		Labels: it.labels(), Milestone: it.milestone(), Body: it.Body,
	}
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/issues/%d/comments?page=%d&limit=50", c.cfg.Repo, n, page))
		if err != nil {
			return forge.IssueInfo{}, err
		}
		var rows []struct {
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			Created string `json:"created_at"`
			Body    string `json:"body"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return forge.IssueInfo{}, fmt.Errorf("the server did not return comments for issue %d", n)
		}
		for _, r := range rows {
			out.Comments = append(out.Comments, forge.Comment{Author: r.User.Login, Created: r.Created, Body: r.Body})
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}
