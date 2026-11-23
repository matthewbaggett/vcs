package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Kind names a forge.
type Kind string

// The forges that Detect knows.
const (
	KindForgejo          Kind = "forgejo"
	KindGitea            Kind = "gitea"
	KindGitHub           Kind = "github"
	KindGitHubEnterprise Kind = "github-enterprise"
)

// Detect finds the forge of a host. The first hit wins:
//  1. github.com is GitHub.
//  2. GET /api/v1/version answers: Forgejo or Gitea. A version with +gitea- is Gitea. A version
//     that names forgejo is Forgejo. A plain version is Gitea.
//  3. GET /api/v3/meta answers: GitHub Enterprise.
//
// When nothing answers, the error names each address that it tried.
func Detect(ctx context.Context, hc *http.Client, host string) (Kind, error) {
	if host == "github.com" {
		return KindGitHub, nil
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	var tried []string
	get := func(path string) ([]byte, bool) {
		u := "https://" + host + path
		tried = append(tried, u)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, false
		}
		resp, err := hc.Do(req)
		if err != nil {
			return nil, false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, false
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return b, err == nil
	}
	if b, ok := get("/api/v1/version"); ok {
		var v struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(b, &v) == nil && v.Version != "" {
			if strings.Contains(strings.ToLower(v.Version), "forgejo") {
				return KindForgejo, nil
			}
			return KindGitea, nil
		}
	}
	if b, ok := get("/api/v3/meta"); ok && json.Valid(b) {
		return KindGitHubEnterprise, nil
	}
	return "", fmt.Errorf("no forge answered on %s. Tried: %s", host, strings.Join(tried, ", "))
}
