package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func (c *Client) packageOwner() (owner, repo string) {
	owner, repo, _ = strings.Cut(c.cfg.Repo, "/")
	return owner, repo
}

// PackageList lists the generic packages of the owner of the repository. The search of the server
// matches part of a name, so the list keeps an exact match only when a name is given.
func (c *Client) PackageList(ctx context.Context, name string) ([]forge.Package, error) {
	owner, _ := c.packageOwner()
	var out []forge.Package
	for page := 1; page <= 100; page++ {
		q := fmt.Sprintf("packages/%s?type=generic&limit=50&page=%d", url.PathEscape(owner), page)
		if name != "" {
			q += "&q=" + url.QueryEscape(name)
		}
		body, err := c.get(ctx, q)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Version string `json:"version"`
			Created string `json:"created_at"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of packages")
		}
		for _, r := range rows {
			if r.Type == "generic" && (name == "" || r.Name == name) {
				out = append(out, forge.Package{Name: r.Name, Version: r.Version, Created: r.Created})
			}
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}

// PackageURL gives the download address of one file of one version.
func (c *Client) PackageURL(name, version, file string) string {
	owner, _ := c.packageOwner()
	return fmt.Sprintf("%s/api/packages/%s/generic/%s/%s/%s", c.cfg.BaseURL, url.PathEscape(owner), url.PathEscape(name), url.PathEscape(version), url.PathEscape(file))
}

// PackageUpload sends one file with a PUT. It makes one attempt and never retries, because a retry
// after a slow answer could race the first upload.
func (c *Client) PackageUpload(ctx context.Context, name, version, file string, r io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.PackageURL(name, version, file), r)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	c.authorize(req)
	_, err = c.send(req, "packages upload")
	return err
}

// PackageLink links a package to the repository.
func (c *Client) PackageLink(ctx context.Context, name string) error {
	owner, repo := c.packageOwner()
	_, err := c.sendJSON(ctx, "POST", fmt.Sprintf("packages/%s/generic/%s/-/link/%s", url.PathEscape(owner), url.PathEscape(name), url.PathEscape(repo)), nil)
	return err
}

// PackageDelete removes one version.
func (c *Client) PackageDelete(ctx context.Context, name, version string) error {
	owner, _ := c.packageOwner()
	_, err := c.sendJSON(ctx, "DELETE", fmt.Sprintf("packages/%s/generic/%s/%s", url.PathEscape(owner), url.PathEscape(name), url.PathEscape(version)), nil)
	return err
}
