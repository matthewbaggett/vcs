package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

var _ forge.CommentManager = (*Client)(nil)

// CommentList reads every comment of an issue or a pull request.
func (c *Client) CommentList(ctx context.Context, n int) ([]forge.CommentRow, error) {
	var out []forge.CommentRow
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/issues/%d/comments?per_page=50&page=%d", c.cfg.Repo, n, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			ID   int64 `json:"id"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			Created string `json:"created_at"`
			Updated string `json:"updated_at"`
			Body    string `json:"body"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return comments for %d", n)
		}
		for _, r := range rows {
			out = append(out, forge.CommentRow{ID: r.ID, Author: r.User.Login, Created: r.Created, Updated: r.Updated, Body: r.Body})
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}

// CommentEdit replaces the text of one comment.
func (c *Client) CommentEdit(ctx context.Context, id int64, body string) error {
	_, err := c.send(ctx, http.MethodPatch, fmt.Sprintf("%s/repos/%s/issues/comments/%d", c.cfg.BaseURL, c.cfg.Repo, id), map[string]string{"body": body})
	return err
}

// CommentDelete removes one comment.
func (c *Client) CommentDelete(ctx context.Context, id int64) error {
	_, err := c.send(ctx, http.MethodDelete, fmt.Sprintf("%s/repos/%s/issues/comments/%d", c.cfg.BaseURL, c.cfg.Repo, id), nil)
	return err
}
