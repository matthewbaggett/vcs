// Package github is the driver for GitHub. It reads with REST calls.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// DefaultBaseURL is the API root of github.com.
const DefaultBaseURL = "https://api.github.com"

// Config sets up a driver.
type Config struct {
	BaseURL string // API root. Default https://api.github.com. GitHub Enterprise uses https://host/api/v3
	Repo    string // owner/name
	Token   string // from GITHUB_TOKEN or GH_TOKEN. Empty means no auth, and a low rate limit
	HTTP    *http.Client
}

// Client is the GitHub driver.
type Client struct {
	cfg Config
}

var _ forge.Client = (*Client)(nil)

// New returns a driver.
func New(cfg Config) *Client {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg}
}

// TokenFromEnv reads the token from GITHUB_TOKEN, then GH_TOKEN. It reads no file.
func TokenFromEnv(getenv func(string) string) string {
	if t := getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	return getenv("GH_TOKEN")
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	body, _, err := c.fetch(ctx, c.cfg.HTTP, c.cfg.BaseURL+"/"+path, true)
	return body, err
}

// fetch makes one call. It returns the Location of a redirect that the client did not follow.
func (c *Client) fetch(ctx context.Context, hc *http.Client, url string, auth bool) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if auth && c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	path := strings.TrimPrefix(url, c.cfg.BaseURL+"/")
	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400 && resp.Header.Get("Location") != "":
		return nil, resp.Header.Get("Location"), nil
	case resp.StatusCode == http.StatusNotFound:
		return nil, "", fmt.Errorf("%s: %w", path, forge.ErrNotFound)
	case resp.StatusCode >= 300:
		return nil, "", &forge.Error{Status: resp.StatusCode, Path: path, Body: string(body)}
	}
	return body, "", nil
}

type prJSON struct {
	State     string `json:"state"`
	Merged    bool   `json:"merged"`
	Draft     bool   `json:"draft"`
	Mergeable *bool  `json:"mergeable"`
	HTMLURL   string `json:"html_url"`
	Head      struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// PRStatus reads one pull request and the CI state of its head commit.
// GitHub does not list the conflicted files, so ConflictedFiles stays empty.
func (c *Client) PRStatus(ctx context.Context, n int) (forge.PRStatus, error) {
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n))
	if err != nil {
		return forge.PRStatus{}, err
	}
	var pr prJSON
	if err := json.Unmarshal(body, &pr); err != nil || pr.Head.SHA == "" {
		return forge.PRStatus{}, fmt.Errorf("the server did not return pull request %d", n)
	}
	ci, err := c.ciSummary(ctx, pr.Head.SHA)
	if err != nil {
		return forge.PRStatus{}, err
	}
	state := pr.State
	if pr.Merged {
		state = "merged"
	}
	return forge.PRStatus{
		State: state, Draft: pr.Draft,
		Mergeable: pr.Mergeable != nil && *pr.Mergeable,
		CI:        ci, HeadSHA: pr.Head.SHA, URL: pr.HTMLURL,
	}, nil
}

type checkRunJSON struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// ciSummary joins the check runs and the legacy commit statuses of one commit, and takes the worst.
func (c *Client) ciSummary(ctx context.Context, sha string) (string, error) {
	var states []string
	newest := map[string]checkRunJSON{}
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/commits/%s/check-runs?per_page=100&page=%d", c.cfg.Repo, sha, page))
		if err != nil {
			return "", err
		}
		var out struct {
			Runs []checkRunJSON `json:"check_runs"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return "", fmt.Errorf("the server did not return check runs")
		}
		for _, r := range out.Runs {
			if old, ok := newest[r.Name]; !ok || r.ID > old.ID {
				newest[r.Name] = r
			}
		}
		if len(out.Runs) < 100 {
			break
		}
	}
	for _, r := range newest {
		states = append(states, checkState(r.Status, r.Conclusion))
	}
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/commits/%s/status", c.cfg.Repo, sha))
	if err != nil {
		return "", err
	}
	var comb struct {
		Statuses []struct {
			State string `json:"state"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal(body, &comb); err != nil {
		return "", fmt.Errorf("the server did not return commit statuses")
	}
	for _, s := range comb.Statuses {
		if s.State == "error" {
			s.State = "failure"
		}
		states = append(states, s.State)
	}
	return summarize(states), nil
}

// checkState maps a check run to failure, pending, warning, or success.
func checkState(status, conclusion string) string {
	if status != "completed" {
		return "pending"
	}
	switch conclusion {
	case "failure", "timed_out", "startup_failure", "action_required":
		return "failure"
	case "neutral":
		return "warning"
	}
	return "success" // success, skipped, and cancelled do not mark the code as bad
}

func summarize(states []string) string {
	if len(states) == 0 {
		return "none"
	}
	worst := "success"
	rank := map[string]int{"success": 0, "warning": 1, "pending": 2, "failure": 3}
	for _, s := range states {
		if rank[s] > rank[worst] {
			worst = s
		}
	}
	return worst
}

// runStatus maps a GitHub status and conclusion to the words of the Forgejo driver.
func runStatus(st, conclusion string) string {
	switch st {
	case "completed":
		switch conclusion {
		case "timed_out", "startup_failure":
			return "failure"
		case "":
			return "success"
		}
		return conclusion // success, failure, cancelled, skipped, neutral, action_required
	case "in_progress":
		return "running"
	case "queued", "requested", "waiting", "pending":
		return "waiting"
	}
	return st
}

type runJSON struct {
	ID           int64  `json:"id"`
	RunNumber    int64  `json:"run_number"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	HeadBranch   string `json:"head_branch"`
	HeadSHA      string `json:"head_sha"`
	DisplayTitle string `json:"display_title"`
	HTMLURL      string `json:"html_url"`
	Event        string `json:"event"`
	PullRequests []struct {
		Number int `json:"number"`
	} `json:"pull_requests"`
}

// ref gives "#12" for a pull request run and the branch for any other run, like the Forgejo driver.
func (r runJSON) ref() string {
	if strings.HasPrefix(r.Event, "pull_request") && len(r.PullRequests) > 0 {
		return fmt.Sprintf("#%d", r.PullRequests[0].Number)
	}
	return r.HeadBranch
}

// CIRuns lists runs, newest first. This side applies the filters on the mapped words.
func (c *Client) CIRuns(ctx context.Context, f forge.RunFilter) ([]forge.Run, error) {
	extra := ""
	if len(f.SHA) == 40 {
		extra = "&head_sha=" + f.SHA
	}
	var kept []forge.Run
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/runs?per_page=100&page=%d%s", c.cfg.Repo, page, extra))
		if err != nil {
			return nil, err
		}
		var out struct {
			Runs []runJSON `json:"workflow_runs"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("the server did not return a run list")
		}
		for _, r := range out.Runs {
			st := runStatus(r.Status, r.Conclusion)
			if f.Branch != "" && r.ref() != f.Branch && r.HeadBranch != f.Branch {
				continue
			}
			if f.SHA != "" && !strings.HasPrefix(r.HeadSHA, f.SHA) {
				continue
			}
			if f.Status != "" && st != f.Status {
				continue
			}
			kept = append(kept, forge.Run{ID: r.ID, Index: r.RunNumber, Status: st, Ref: r.ref(), SHA: r.HeadSHA, Title: r.DisplayTitle, URL: r.HTMLURL})
		}
		if len(out.Runs) < 100 || (f.Limit > 0 && len(kept) >= f.Limit) || len(kept) >= forge.MaxRuns {
			break
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].ID > kept[j].ID })
	max := f.Limit
	if max == 0 || max > forge.MaxRuns {
		max = forge.MaxRuns
	}
	if len(kept) > max {
		kept = kept[:max]
	}
	return kept, nil
}

type jobJSON struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Attempt    int    `json:"run_attempt"`
}

// CIJobs lists the jobs of one run, oldest first.
func (c *Client) CIJobs(ctx context.Context, runID int64) ([]forge.Job, error) {
	var rows []jobJSON
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/runs/%d/jobs?per_page=100&page=%d", c.cfg.Repo, runID, page))
		if err != nil {
			return nil, err
		}
		var out struct {
			Jobs []jobJSON `json:"jobs"`
		}
		if err := json.Unmarshal(body, &out); err != nil || out.Jobs == nil {
			return nil, fmt.Errorf("the server did not return a job list for run %d", runID)
		}
		rows = append(rows, out.Jobs...)
		if len(out.Jobs) < 100 {
			break
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	jobs := make([]forge.Job, 0, len(rows))
	for _, r := range rows {
		a := r.Attempt
		if a == 0 {
			a = 1
		}
		st := runStatus(r.Status, r.Conclusion)
		jobs = append(jobs, forge.Job{ID: r.ID, Name: r.Name, Status: st, Attempt: a, NoLog: st == "skipped"})
	}
	return jobs, nil
}

// CILog reads the log of one job. GitHub answers 302 with a signed link. This side fetches that
// link without the token, so the token never reaches the storage host. 410 means the log expired.
func (c *Client) CILog(ctx context.Context, jobID int64) (string, error) {
	path := fmt.Sprintf("repos/%s/actions/jobs/%d/logs", c.cfg.Repo, jobID)
	noFollow := *c.cfg.HTTP
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	body, loc, err := c.fetch(ctx, &noFollow, c.cfg.BaseURL+"/"+path, true)
	var fe *forge.Error
	if errors.As(err, &fe) && fe.Status == http.StatusGone {
		return "", forge.ErrNoLog
	}
	if err != nil {
		return "", err
	}
	if loc != "" {
		body, _, err = c.fetch(ctx, c.cfg.HTTP, loc, false)
		if err != nil {
			return "", err
		}
	}
	return string(body), nil
}
