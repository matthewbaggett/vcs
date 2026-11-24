package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// SecretList reads every Actions secret name of the repository. The server never returns a value.
func (c *Client) SecretList(ctx context.Context) ([]forge.Secret, error) {
	var out []forge.Secret
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/secrets?limit=50&page=%d", c.cfg.Repo, page))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Name    string `json:"name"`
			Created string `json:"created_at"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of secrets")
		}
		for _, r := range rows {
			out = append(out, forge.Secret{Name: r.Name, Created: r.Created})
		}
		if len(rows) < 50 {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SecretSet creates or replaces one secret. The error text never holds the value.
func (c *Client) SecretSet(ctx context.Context, name, value string) error {
	_, err := c.sendJSON(ctx, "PUT", fmt.Sprintf("repos/%s/actions/secrets/%s", c.cfg.Repo, url.PathEscape(name)), map[string]string{"data": value})
	return err
}

// SecretDelete removes one secret.
func (c *Client) SecretDelete(ctx context.Context, name string) error {
	_, err := c.sendJSON(ctx, "DELETE", fmt.Sprintf("repos/%s/actions/secrets/%s", c.cfg.Repo, url.PathEscape(name)), nil)
	return err
}
