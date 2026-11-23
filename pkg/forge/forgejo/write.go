package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func (c *Client) authorize(req *http.Request) {
	switch {
	case c.cfg.Token != "":
		req.Header.Set("Authorization", "token "+c.cfg.Token)
	case c.cfg.Login != "":
		req.SetBasicAuth(c.cfg.Login, c.cfg.Password)
	}
}

func (c *Client) send(req *http.Request, path string) ([]byte, error) {
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
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusSeeOther {
		return nil, &forge.Error{Status: resp.StatusCode, Path: path, Body: string(body)}
	}
	return body, nil
}

// webPost posts to a web route with the session cookie. These routes refuse a token.
func (c *Client) webPost(ctx context.Context, path string) error {
	if c.cfg.Cookie == "" {
		return forge.ErrNoSession
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "i_like_gitea", Value: c.cfg.Cookie})
	// Forgejo checks these two headers on a web write. A same-origin value passes.
	req.Header.Set("Origin", c.cfg.BaseURL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	hc := *c.cfg.HTTP
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch {
	case resp.StatusCode == http.StatusSeeOther && strings.Contains(resp.Header.Get("Location"), "/user/login"):
		return forge.ErrNoSession
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%s: %w", path, forge.ErrNotFound)
	case resp.StatusCode >= 400:
		return &forge.Error{Status: resp.StatusCode, Path: path, Body: string(body)}
	}
	return nil
}

// CIRerun restarts a run through the web route. The Forgejo API has no rerun route.
// The web routes take the run index and, for one job, the position of the job in the run.
func (c *Client) CIRerun(ctx context.Context, runID int64, failedOnly bool) error {
	runs, err := c.CIRuns(ctx, forge.RunFilter{Limit: 250})
	if err != nil {
		return err
	}
	var run *forge.Run
	for i := range runs {
		if runs[i].ID == runID {
			run = &runs[i]
		}
	}
	if run == nil {
		return fmt.Errorf("run %d is not in the newest 250 runs: %w", runID, forge.ErrNotFound)
	}
	base := fmt.Sprintf("/%s/actions/runs/%d", c.cfg.Repo, run.Index)
	if !failedOnly {
		return c.webPost(ctx, base+"/rerun")
	}
	jobs, err := c.CIJobs(ctx, runID)
	if err != nil {
		return err
	}
	// The server lists jobs by id, which is also their order in the run.
	n := 0
	for i, j := range jobs {
		if j.Status == "failure" || j.Status == "cancelled" {
			if err := c.webPost(ctx, fmt.Sprintf("%s/jobs/%d/rerun", base, i)); err != nil {
				return err
			}
			n++
		}
	}
	if n == 0 {
		return fmt.Errorf("run %d has no failed job", runID)
	}
	return nil
}

// Attach uploads each file with the asset route and returns the download addresses.
func (c *Client) Attach(ctx context.Context, n int, files []string) ([]forge.Asset, error) {
	var out []forge.Asset
	for _, f := range files {
		name := filepath.Base(f)
		data, err := os.ReadFile(f)
		if err != nil {
			return out, err
		}
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		part, err := mw.CreateFormFile("attachment", name)
		if err != nil {
			return out, err
		}
		part.Write(data)
		mw.Close()
		path := fmt.Sprintf("repos/%s/issues/%d/assets?name=%s", c.cfg.Repo, n, url.QueryEscape(name))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, &buf)
		if err != nil {
			return out, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		c.authorize(req)
		body, err := c.send(req, path)
		if err != nil {
			return out, fmt.Errorf("the upload of %s failed: %w", name, err)
		}
		var a struct {
			URL string `json:"browser_download_url"`
		}
		if json.Unmarshal(body, &a) != nil || a.URL == "" {
			return out, fmt.Errorf("no download address for %s", name)
		}
		out = append(out, forge.Asset{Name: name, URL: a.URL})
	}
	return out, nil
}

// Comment posts one comment.
func (c *Client) Comment(ctx context.Context, n int, body string) error {
	payload, _ := json.Marshal(map[string]string{"body": body})
	path := fmt.Sprintf("repos/%s/issues/%d/comments", c.cfg.Repo, n)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	_, err = c.send(req, path)
	return err
}

// CreatePR opens a pull request. A draft carries the WIP: prefix, which Forgejo reads as a draft.
func (c *Client) CreatePR(ctx context.Context, in forge.NewPR) (int, string, error) {
	title := in.Title
	if in.Draft && !strings.HasPrefix(strings.ToUpper(title), "WIP:") {
		title = "WIP: " + title
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": in.Body, "head": in.Head, "base": in.Base})
	path := fmt.Sprintf("repos/%s/pulls", c.cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	body, err := c.send(req, path)
	if err != nil {
		return 0, "", err
	}
	var out struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	if json.Unmarshal(body, &out) != nil || out.Number == 0 {
		return 0, "", fmt.Errorf("the server gave no pull request number")
	}
	return out.Number, out.URL, nil
}

// CreateIssue opens an issue.
func (c *Client) CreateIssue(ctx context.Context, title, body string) (int, string, error) {
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body})
	path := fmt.Sprintf("repos/%s/issues", c.cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	resp, err := c.send(req, path)
	if err != nil {
		return 0, "", err
	}
	var out struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	if json.Unmarshal(resp, &out) != nil || out.Number == 0 {
		return 0, "", fmt.Errorf("the server gave no issue number")
	}
	return out.Number, out.URL, nil
}

// PREdit changes the title or the body of a pull request. A draft keeps its WIP: prefix,
// because Forgejo reads the prefix as the draft state.
func (c *Client) PREdit(ctx context.Context, n int, in forge.PREdit) error {
	path := fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n)
	payload := map[string]string{}
	if in.Body != nil {
		payload["body"] = *in.Body
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		get, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/"+path, nil)
		if err != nil {
			return err
		}
		c.authorize(get)
		resp, err := c.send(get, path)
		if err != nil {
			return err
		}
		var pr struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(resp, &pr); err != nil {
			return err
		}
		if up := strings.ToUpper(strings.TrimSpace(pr.Title)); strings.HasPrefix(up, "WIP:") && !strings.HasPrefix(strings.ToUpper(title), "WIP:") {
			title = "WIP: " + title
		}
		payload["title"] = title
	}
	if len(payload) == 0 {
		return nil
	}
	raw, _ := json.Marshal(payload)
	patch, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	patch.Header.Set("Content-Type", "application/json")
	c.authorize(patch)
	_, err = c.send(patch, path)
	return err
}

// PRReady removes the WIP: prefix from the title, which takes the pull request out of draft.
func (c *Client) PRReady(ctx context.Context, n int) error {
	path := fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n)
	get, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/"+path, nil)
	if err != nil {
		return err
	}
	c.authorize(get)
	resp, err := c.send(get, path)
	if err != nil {
		return err
	}
	var pr struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(resp, &pr); err != nil {
		return err
	}
	title := strings.TrimSpace(pr.Title)
	up := strings.ToUpper(title)
	switch {
	case strings.HasPrefix(up, "WIP:"):
		title = strings.TrimSpace(title[4:])
	case strings.HasPrefix(up, "[WIP]"):
		title = strings.TrimSpace(title[5:])
	default:
		return nil // already ready
	}
	payload, _ := json.Marshal(map[string]string{"title": title})
	patch, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.cfg.BaseURL+"/api/v1/"+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	patch.Header.Set("Content-Type", "application/json")
	c.authorize(patch)
	_, err = c.send(patch, path)
	return err
}
