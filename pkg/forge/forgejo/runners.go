package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// RunnerList reads every CI runner of the instance. The call is GET /admin/actions/runners. It
// needs a token of an admin user. The repository and organization forms of the call return an
// empty list for a runner that the whole instance shares, so this verb does not use them. A 401
// or 403 means the token is missing, too weak, or not an admin, and gives ErrNoSession.
func (c *Client) RunnerList(ctx context.Context) ([]forge.Runner, error) {
	var out []forge.Runner
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("admin/actions/runners?limit=50&page=%d", page))
		if err != nil {
			var fe *forge.Error
			if errors.As(err, &fe) && (fe.Status == 401 || fe.Status == 403) {
				return nil, fmt.Errorf("%w. The server refused the token: %s", forge.ErrNoSession, fe.Body)
			}
			return nil, err
		}
		var rows []struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			UUID    string `json:"uuid"`
			Version string `json:"version"`
			Labels  []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of runners")
		}
		for _, r := range rows {
			labels := make([]string, 0, len(r.Labels))
			for _, l := range r.Labels {
				labels = append(labels, l.Name)
			}
			out = append(out, forge.Runner{ID: r.ID, Name: r.Name, Status: r.Status, Labels: labels, UUID: r.UUID, Version: r.Version})
		}
		if len(rows) < 50 {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
