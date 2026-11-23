package forge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRemote(t *testing.T) {
	good := map[string][2]string{
		"https://git.grey.ooo/packages/vcs.git":       {"git.grey.ooo", "packages/vcs"},
		"https://user@git.grey.ooo:8443/packages/vcs": {"git.grey.ooo", "packages/vcs"},
		"git@github.com:matthewbaggett/vcs.git":       {"github.com", "matthewbaggett/vcs"},
		"ssh://git@git.grey.ooo/packages/vcs.git":     {"git.grey.ooo", "packages/vcs"},
	}
	for in, want := range good {
		h, r, err := ParseRemote(in)
		if err != nil || h != want[0] || r != want[1] {
			t.Errorf("%s -> %s %s %v", in, h, r, err)
		}
	}
	for _, in := range []string{"", "nonsense", "https://host/onlyone"} {
		if _, _, err := ParseRemote(in); err == nil {
			t.Errorf("%q should fail", in)
		}
	}
}

func TestParseNetrc(t *testing.T) {
	data := "machine a.example login x password y\nmachine git.example\n  login me\n  password secret\n"
	l, p := parseNetrc(data, "git.example")
	if l != "me" || p != "secret" {
		t.Errorf("got %q %q", l, p)
	}
	if l, p := parseNetrc(data, "none.example"); l != "" || p != "" {
		t.Errorf("got %q %q", l, p)
	}
}

func TestDetect(t *testing.T) {
	for name, tc := range map[string]struct {
		routes map[string]string
		want   Kind
	}{
		"forgejo":    {map[string]string{"/api/v1/version": `{"version":"11.0.0+gitea-1.22.0 (forgejo)"}`}, KindForgejo},
		"forgejo v2": {map[string]string{"/api/v1/version": `{"version":"9.0.0-forgejo"}`}, KindForgejo},
		"gitea":      {map[string]string{"/api/v1/version": `{"version":"1.22.0"}`}, KindGitea},
		"enterprise": {map[string]string{"/api/v3/meta": `{"verifiable_password_authentication":true}`}, KindGitHubEnterprise},
	} {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if body, ok := tc.routes[r.URL.Path]; ok {
				w.Write([]byte(body))
				return
			}
			http.NotFound(w, r)
		}))
		host := strings.TrimPrefix(srv.URL, "https://")
		got, err := Detect(context.Background(), srv.Client(), host)
		srv.Close()
		if err != nil || got != tc.want {
			t.Errorf("%s: got %q err=%v", name, got, err)
		}
	}
	if k, err := Detect(context.Background(), nil, "github.com"); err != nil || k != KindGitHub {
		t.Errorf("github.com: %q %v", k, err)
	}
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "https://")
	if _, err := Detect(context.Background(), srv.Client(), host); err == nil || !strings.Contains(err.Error(), "/api/v1/version") || !strings.Contains(err.Error(), "/api/v3/meta") {
		t.Errorf("none: %v", err)
	}
}
