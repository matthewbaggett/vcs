package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type artifactJSON struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Size    int64  `json:"size_in_bytes"`
	Expired bool   `json:"expired"`
}

// CIArtifacts lists the artifacts of one run with the API route. It needs no web session.
// Gitea has no such route. A 404 from a server that is not Forgejo means unsupported.
func (c *Client) CIArtifacts(ctx context.Context, runID int64) ([]forge.Artifact, error) {
	var out []forge.Artifact
	for page := 1; page <= 20; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/actions/runs/%d/artifacts?page=%d&limit=50", c.cfg.Repo, runID, page))
		if err != nil {
			if errorsIsNotFound(err) && !c.isForgejo(ctx) {
				return nil, fmt.Errorf("this server has no artifact route: %w", forge.ErrUnsupported)
			}
			return nil, err
		}
		var rows struct {
			Artifacts []artifactJSON `json:"artifacts"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			// An older build sends a bare array.
			var bare []artifactJSON
			if err2 := json.Unmarshal(body, &bare); err2 != nil {
				return nil, fmt.Errorf("the server did not return an artifact list for run %d", runID)
			}
			rows.Artifacts = bare
		}
		for _, r := range rows.Artifacts {
			out = append(out, forge.Artifact{ID: r.ID, Name: r.Name, Size: r.Size, Expired: r.Expired})
		}
		if len(rows.Artifacts) < 50 {
			break
		}
	}
	return out, nil
}

func errorsIsNotFound(err error) bool {
	for e := err; e != nil; {
		if e == forge.ErrNotFound {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// isForgejo reads the version. Forgejo says `+gitea-` in it, and Gitea does not.
func (c *Client) isForgejo(ctx context.Context) bool {
	body, err := c.get(ctx, "version")
	if err != nil {
		return true // when in doubt, report the 404 as it came
	}
	var v struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(body, &v) != nil {
		return true
	}
	return strings.Contains(v.Version, "+gitea-")
}

// CIArtifactGet streams the zip archive of one artifact to w.
func (c *Client) CIArtifactGet(ctx context.Context, a forge.Artifact, w io.Writer) (int64, error) {
	path := fmt.Sprintf("repos/%s/actions/artifacts/%d/zip", c.cfg.Repo, a.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/"+path, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return 0, fmt.Errorf("%s: %w", path, forge.ErrNotFound)
	case resp.StatusCode >= 300:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return 0, &forge.Error{Status: resp.StatusCode, Path: path, Body: string(b)}
	}
	return io.Copy(w, resp.Body)
}
