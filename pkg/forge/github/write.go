package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// send makes one call with a JSON body and returns the answer body.
func (c *Client) send(ctx context.Context, method, url string, payload any) ([]byte, error) {
	var rd io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
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
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%s: %w", url, forge.ErrNotFound)
	case resp.StatusCode >= 300:
		return nil, &forge.Error{Status: resp.StatusCode, Path: url, Body: string(body)}
	}
	return body, nil
}

func (c *Client) post(ctx context.Context, path string, payload any) ([]byte, error) {
	return c.send(ctx, http.MethodPost, c.cfg.BaseURL+"/"+path, payload)
}

// CIRerun restarts a run. With failedOnly it restarts the failed jobs only.
func (c *Client) CIRerun(ctx context.Context, runID int64, failedOnly bool) error {
	verb := "rerun"
	if failedOnly {
		verb = "rerun-failed-jobs"
	}
	_, err := c.post(ctx, fmt.Sprintf("repos/%s/actions/runs/%d/%s", c.cfg.Repo, runID, verb), nil)
	return err
}

// Attach is not possible. GitHub has no public upload route for issue files.
func (c *Client) Attach(ctx context.Context, n int, files []string) ([]forge.Asset, error) {
	return nil, forge.ErrUnsupported
}

// Comment posts one comment on an issue or a pull request.
func (c *Client) Comment(ctx context.Context, n int, body string) (int64, string, error) {
	resp, err := c.post(ctx, fmt.Sprintf("repos/%s/issues/%d/comments", c.cfg.Repo, n), map[string]string{"body": body})
	if err != nil {
		return 0, "", err
	}
	var out struct {
		ID      int64  `json:"id"`
		HTMLURL string `json:"html_url"`
	}
	if json.Unmarshal(resp, &out) != nil || out.ID == 0 {
		return 0, "", fmt.Errorf("the server did not return the new comment")
	}
	return out.ID, out.HTMLURL, nil
}

// PREdit changes the title, the body or the base branch of a pull request.
func (c *Client) PREdit(ctx context.Context, n int, in forge.PREdit) error {
	payload := map[string]string{}
	if in.Title != nil {
		payload["title"] = *in.Title
	}
	if in.Body != nil {
		payload["body"] = *in.Body
	}
	if in.Base != nil {
		payload["base"] = *in.Base
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := c.send(ctx, http.MethodPatch, c.cfg.BaseURL+"/"+fmt.Sprintf("repos/%s/pulls/%d", c.cfg.Repo, n), payload)
	return err
}

type created struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

func parseCreated(body []byte, what string) (int, string, error) {
	var v created
	if json.Unmarshal(body, &v) != nil || v.Number == 0 {
		return 0, "", fmt.Errorf("the server did not return the new %s", what)
	}
	return v.Number, v.HTMLURL, nil
}

// CreatePR opens a pull request. The draft field makes it a draft.
func (c *Client) CreatePR(ctx context.Context, in forge.NewPR) (int, string, error) {
	body, err := c.post(ctx, fmt.Sprintf("repos/%s/pulls", c.cfg.Repo), map[string]any{
		"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base, "draft": in.Draft,
	})
	if err != nil {
		return 0, "", err
	}
	return parseCreated(body, "pull request")
}

// CreateIssue opens an issue.
func (c *Client) CreateIssue(ctx context.Context, title, body string) (int, string, error) {
	out, err := c.post(ctx, fmt.Sprintf("repos/%s/issues", c.cfg.Repo), map[string]string{"title": title, "body": body})
	if err != nil {
		return 0, "", err
	}
	return parseCreated(out, "issue")
}

// PRReady takes a pull request out of draft. REST has no route for it, so this uses GraphQL.
func (c *Client) PRReady(ctx context.Context, n int) error {
	var q struct {
		Repository struct {
			PullRequest struct {
				ID      string `json:"id"`
				IsDraft bool   `json:"isDraft"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}
	if err := c.gql(ctx, `query($owner:String!,$name:String!,$n:Int!){repository(owner:$owner,name:$name){pullRequest(number:$n){id isDraft}}}`,
		c.repoVars(map[string]any{"n": n}), &q); err != nil {
		return err
	}
	pr := q.Repository.PullRequest
	if pr.ID == "" {
		return fmt.Errorf("pull request %d: %w", n, forge.ErrNotFound)
	}
	if !pr.IsDraft {
		return nil
	}
	return c.gql(ctx, `mutation($id:ID!){markPullRequestReadyForReview(input:{pullRequestId:$id}){pullRequest{id}}}`,
		map[string]any{"id": pr.ID}, nil)
}
