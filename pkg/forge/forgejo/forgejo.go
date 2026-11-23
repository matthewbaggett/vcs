// Package forgejo is the driver for Forgejo and Gitea. It reads with token calls.
package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// Config sets up a driver.
type Config struct {
	BaseURL    string // for example https://git.grey.ooo
	Repo       string // owner/name
	Token      string // an API token. Empty means use Login and Password
	Login      string
	Password   string
	HTTP       *http.Client
	RetrySleep time.Duration // wait between log tries. Default 2s
}

// Client is the Forgejo driver.
type Client struct {
	cfg Config
}

var _ forge.Client = (*Client)(nil)

// New returns a driver.
func New(cfg Config) *Client {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.RetrySleep == 0 {
		cfg.RetrySleep = 2 * time.Second
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg}
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/"+path, nil)
	if err != nil {
		return nil, err
	}
	switch {
	case c.cfg.Token != "":
		req.Header.Set("Authorization", "token "+c.cfg.Token)
	case c.cfg.Login != "":
		req.SetBasicAuth(c.cfg.Login, c.cfg.Password)
	}
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: %w", path, forge.ErrNotFound)
	}
	if resp.StatusCode >= 300 {
		return nil, &forge.Error{Status: resp.StatusCode, Path: path, Body: string(body)}
	}
	return body, nil
}

type prJSON struct {
	State           string   `json:"state"`
	Merged          bool     `json:"merged"`
	Draft           bool     `json:"draft"`
	Mergeable       bool     `json:"mergeable"`
	ConflictedFiles []string `json:"conflicted_files"`
	HTMLURL         string   `json:"html_url"`
	Head            struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// PRStatus reads one pull request and the CI state of its head commit.
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
		State: state, Draft: pr.Draft, Mergeable: pr.Mergeable,
		ConflictedFiles: pr.ConflictedFiles, CI: ci, HeadSHA: pr.Head.SHA, URL: pr.HTMLURL,
	}, nil
}

type statusJSON struct {
	ID      int64  `json:"id"`
	Context string `json:"context"`
	Status  string `json:"status"`
}

// ciSummary reads every commit status, keeps the newest row of each context, and takes the worst.
func (c *Client) ciSummary(ctx context.Context, sha string) (string, error) {
	newest := map[string]statusJSON{}
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/commits/%s/statuses?page=%d&limit=50", c.cfg.Repo, sha, page))
		if err != nil {
			return "", err
		}
		var rows []statusJSON
		if err := json.Unmarshal(body, &rows); err != nil {
			return "", fmt.Errorf("the server did not return statuses")
		}
		for _, r := range rows {
			if old, ok := newest[r.Context]; !ok || r.ID > old.ID {
				newest[r.Context] = r
			}
		}
		if len(rows) < 50 {
			break
		}
	}
	return Summarize(func() []string {
		s := make([]string, 0, len(newest))
		for _, r := range newest {
			s = append(s, r.Status)
		}
		return s
	}()), nil
}

// Summarize gives the worst state of a set of statuses.
func Summarize(statuses []string) string {
	if len(statuses) == 0 {
		return "none"
	}
	has := func(words ...string) bool {
		for _, s := range statuses {
			for _, w := range words {
				if s == w {
					return true
				}
			}
		}
		return false
	}
	switch {
	case has("failure", "error"):
		return "failure"
	case has("pending"):
		return "pending"
	case has("warning"):
		return "warning"
	}
	return "success"
}

type runJSON struct {
	ID      int64  `json:"id"`
	Index   int64  `json:"index_in_repo"`
	Status  string `json:"status"`
	Ref     string `json:"prettyref"`
	SHA     string `json:"commit_sha"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

// CIRuns lists runs, newest first. The server ignores some filters, so this side applies them.
func (c *Client) CIRuns(ctx context.Context, f forge.RunFilter) ([]forge.Run, error) {
	limit := f.Limit
	extra := ""
	if len(f.SHA) == 40 {
		extra = "&head_sha=" + f.SHA
	}
	var kept []forge.Run
	seen := map[int64]bool{}
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/runs?page=%d&limit=50%s", c.cfg.Repo, page, extra))
		if err != nil {
			return nil, err
		}
		var out struct {
			Runs []runJSON `json:"workflow_runs"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("the server did not return a run list")
		}
		if len(out.Runs) == 0 {
			break
		}
		fresh := 0
		for _, r := range out.Runs {
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			fresh++
			if f.Branch != "" && r.Ref != f.Branch {
				continue
			}
			if f.SHA != "" && !strings.HasPrefix(r.SHA, f.SHA) {
				continue
			}
			if f.Status != "" && r.Status != f.Status {
				continue
			}
			kept = append(kept, forge.Run{ID: r.ID, Index: r.Index, Status: r.Status, Ref: r.Ref, SHA: r.SHA, Title: r.Title, URL: r.HTMLURL})
		}
		if fresh == 0 || (limit > 0 && len(kept) >= limit) || len(kept) >= forge.MaxRuns {
			break
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].ID > kept[j].ID })
	max := limit
	if max == 0 || max > forge.MaxRuns {
		max = forge.MaxRuns
	}
	if len(kept) > max {
		kept = kept[:max]
	}
	return kept, nil
}

type jobJSON struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Attempt int      `json:"attempt"`
	RunsOn  []string `json:"runs_on"`
}

// CIJobs lists the jobs of one run. The server sends a bare array, and a newer one may wrap it.
func (c *Client) CIJobs(ctx context.Context, runID int64) ([]forge.Job, error) {
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/runs/%d/jobs", c.cfg.Repo, runID))
	if err != nil {
		return nil, err
	}
	var rows []jobJSON
	if err := json.Unmarshal(body, &rows); err != nil {
		var wrapped struct {
			Jobs []jobJSON `json:"workflow_jobs"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil || wrapped.Jobs == nil {
			return nil, fmt.Errorf("the server did not return a job list for run %d", runID)
		}
		rows = wrapped.Jobs
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	jobs := make([]forge.Job, 0, len(rows))
	for _, r := range rows {
		a := r.Attempt
		if a == 0 {
			a = 1
		}
		jobs = append(jobs, forge.Job{ID: r.ID, Name: r.Name, Status: r.Status, Attempt: a, NoLog: len(r.RunsOn) == 0})
	}
	return jobs, nil
}

// CILog reads the log of one job. A busy server answers 504, so this makes up to three tries.
func (c *Client) CILog(ctx context.Context, jobID int64) (string, error) {
	var last error
	for try := 0; try < 3; try++ {
		if try > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(c.cfg.RetrySleep):
			}
		}
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/jobs/%d/logs", c.cfg.Repo, jobID))
		if err == nil {
			return string(body), nil
		}
		last = err
		var fe *forge.Error
		if asErr(err, &fe) && strings.Contains(fe.Body, "file does not exist") {
			return "", forge.ErrNoLog
		}
	}
	return "", last
}

func asErr(err error, target **forge.Error) bool {
	e, ok := err.(*forge.Error)
	if ok {
		*target = e
	}
	return ok
}
