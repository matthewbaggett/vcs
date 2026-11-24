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
	Cookie     string // the i_like_gitea value of a web session, for the web routes
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
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
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
	ID        int64  `json:"id"`
	Context   string `json:"context"`
	Status    string `json:"status"`
	TargetURL string `json:"target_url"`
}

// newestStatuses reads every commit status and keeps the newest row of each context.
func (c *Client) newestStatuses(ctx context.Context, sha string) ([]statusJSON, error) {
	newest := map[string]statusJSON{}
	var order []string
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/commits/%s/statuses?page=%d&limit=50", c.cfg.Repo, sha, page))
		if err != nil {
			return nil, err
		}
		var rows []statusJSON
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return statuses")
		}
		for _, r := range rows {
			old, ok := newest[r.Context]
			if !ok {
				order = append(order, r.Context)
			}
			if !ok || r.ID > old.ID {
				newest[r.Context] = r
			}
		}
		if len(rows) < 50 {
			break
		}
	}
	out := make([]statusJSON, 0, len(order))
	for _, k := range order {
		out = append(out, newest[k])
	}
	return out, nil
}

// CommitStatuses reads every status row of one commit. A target address that is a path gets the
// host in front of it.
func (c *Client) CommitStatuses(ctx context.Context, sha string) ([]forge.CommitStatus, error) {
	var out []forge.CommitStatus
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/commits/%s/statuses?page=%d&limit=50", c.cfg.Repo, sha, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			statusJSON
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return statuses for %s", sha)
		}
		for _, r := range rows {
			u := r.TargetURL
			if strings.HasPrefix(u, "/") {
				u = c.cfg.BaseURL + u
			}
			out = append(out, forge.CommitStatus{ID: r.ID, Context: r.Context, Status: r.Status, URL: u, Description: r.Description})
		}
		if len(rows) == 0 {
			break
		}
	}
	return out, nil
}

// IssueTimeline reads every event of one issue or pull request, oldest first.
func (c *Client) IssueTimeline(ctx context.Context, n int) ([]forge.TimelineEvent, error) {
	type row struct {
		ID        int64                  `json:"id"`
		Created   string                 `json:"created_at"`
		Type      string                 `json:"type"`
		Body      string                 `json:"body"`
		OldTitle  string                 `json:"old_title"`
		NewTitle  string                 `json:"new_title"`
		RefCommit string                 `json:"ref_commit_sha"`
		User      struct{ Login string } `json:"user"`
		Label     struct{ Name string }  `json:"label"`
	}
	var all []row
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/issues/%d/timeline?page=%d&limit=50", c.cfg.Repo, n, page))
		if err != nil {
			return nil, err
		}
		var rows []row
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return the timeline of %d", n)
		}
		all = append(all, rows...)
		if len(rows) == 0 {
			break
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Created != all[j].Created {
			return all[i].Created < all[j].Created
		}
		return all[i].ID < all[j].ID
	})
	out := make([]forge.TimelineEvent, 0, len(all))
	for _, r := range all {
		d := r.Body
		switch {
		case r.Type == "label":
			sign := "+"
			if r.Body == "" {
				sign = "-"
			}
			d = sign + r.Label.Name
		case r.Type == "change_title":
			d = r.OldTitle + " -> " + r.NewTitle
		case r.Type == "pull_push":
			var p struct {
				Force bool     `json:"is_force_push"`
				IDs   []string `json:"commit_ids"`
			}
			if json.Unmarshal([]byte(r.Body), &p) == nil && (p.IDs != nil || p.Force) {
				word := "push "
				if p.Force {
					word = "force-push "
				}
				d = word + strings.Join(p.IDs, ",")
			}
		case r.Type == "commit_ref" && r.RefCommit != "":
			d = r.RefCommit
		}
		out = append(out, forge.TimelineEvent{ID: r.ID, Created: r.Created, Type: r.Type, User: r.User.Login, Detail: d})
	}
	return out, nil
}

// ciSummary reads every commit status, keeps the newest row of each context, and takes the worst.
func (c *Client) ciSummary(ctx context.Context, sha string) (string, error) {
	rows, err := c.newestStatuses(ctx, sha)
	if err != nil {
		return "", err
	}
	s := make([]string, 0, len(rows))
	for _, r := range rows {
		s = append(s, r.Status)
	}
	return Summarize(s), nil
}

// PRWatch reads the pull request, the newest status of each context, and the required contexts of
// the base branch. A token without admin rights can't read the rule, and an unprotected branch has
// none. Either way no check counts as required.
func (c *Client) PRWatch(ctx context.Context, n int) (forge.WatchState, error) {
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n))
	if err != nil {
		return forge.WatchState{}, err
	}
	var pr prJSON
	if err := json.Unmarshal(body, &pr); err != nil || pr.Head.SHA == "" {
		return forge.WatchState{}, fmt.Errorf("the server did not return pull request %d", n)
	}
	w := forge.WatchState{State: pr.State, HeadSHA: pr.Head.SHA, Base: pr.Base.Ref}
	if pr.Merged {
		w.State = "merged"
	}
	if w.State != "open" {
		return w, nil
	}
	rows, err := c.newestStatuses(ctx, pr.Head.SHA)
	if err != nil {
		return forge.WatchState{}, err
	}
	for _, r := range rows {
		w.Checks = append(w.Checks, forge.Check{Context: r.Context, Status: r.Status, URL: r.TargetURL})
	}
	if raw, err := c.get(ctx, fmt.Sprintf("repos/%s/branch_protections/%s", c.cfg.Repo, pr.Base.Ref)); err == nil {
		var rule struct {
			Contexts []string `json:"status_check_contexts"`
		}
		if json.Unmarshal(raw, &rule) == nil {
			w.Required = rule.Contexts
		}
	}
	return w, nil
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

type commitJSON struct {
	Commit struct {
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

// CommitTime reads the commit time of one commit, in UTC.
func (c *Client) CommitTime(ctx context.Context, sha string) (string, error) {
	body, err := c.get(ctx, fmt.Sprintf("repos/%s/git/commits/%s", c.cfg.Repo, sha))
	if err != nil {
		return "", err
	}
	var cj commitJSON
	if err := json.Unmarshal(body, &cj); err != nil {
		return "", fmt.Errorf("the server did not return commit %s", sha)
	}
	t, err := time.Parse(time.RFC3339, cj.Commit.Committer.Date)
	if err != nil {
		return "", fmt.Errorf("the server gave no commit time for %s", sha)
	}
	return t.UTC().Format("2006-01-02T15:04:05Z"), nil
}
