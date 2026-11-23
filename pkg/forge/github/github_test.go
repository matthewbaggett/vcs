package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func newServer(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok"})
}

func TestPRStatus(t *testing.T) {
	var auth, accept string
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth, accept = r.Header.Get("Authorization"), r.Header.Get("Accept")
		switch r.URL.Path {
		case "/repos/o/r/pulls/12":
			w.Write(fixture(t, "pr12.json"))
		case "/repos/o/r/commits/" + sha + "/check-runs":
			w.Write(fixture(t, "checkruns.json"))
		case "/repos/o/r/commits/" + sha + "/status":
			w.Write(fixture(t, "status.json"))
		default:
			http.NotFound(w, r)
		}
	})
	s, err := c.PRStatus(context.Background(), 12)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer tok" || !strings.Contains(accept, "github") {
		t.Errorf("headers = %q %q", auth, accept)
	}
	// test: the newest row is success. lint is in progress, so the worst is pending.
	if s.State != "open" || !s.Draft || s.Mergeable || s.CI != "pending" || s.HeadSHA != sha || s.URL == "" {
		t.Errorf("status = %+v", s)
	}
	if _, err := c.PRStatus(context.Background(), 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestSummarize(t *testing.T) {
	cases := map[string][]string{
		"none":    nil,
		"failure": {"success", "failure", "pending"},
		"pending": {"success", "pending", "warning"},
		"warning": {"success", "warning"},
		"success": {"success"},
	}
	for want, in := range cases {
		if got := summarize(in); got != want {
			t.Errorf("summarize(%v) = %s, want %s", in, got, want)
		}
	}
}

func TestCIRuns(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture(t, "runs.json"))
	})
	ctx := context.Background()
	runs, err := c.CIRuns(ctx, forge.RunFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || runs[0].ID != 104 || runs[0].Status != "running" ||
		runs[1].Ref != "#12" && runs[2].Ref != "#12" {
		t.Errorf("runs = %+v", runs)
	}
	if runs[1].ID != 103 || runs[1].Status != "failure" || runs[1].Index != 30 || runs[1].Title != "Fix it" {
		t.Errorf("runs[1] = %+v", runs[1])
	}
	runs, _ = c.CIRuns(ctx, forge.RunFilter{Branch: "main", Status: "success"})
	if len(runs) != 1 || runs[0].ID != 102 {
		t.Errorf("filtered = %+v", runs)
	}
	runs, _ = c.CIRuns(ctx, forge.RunFilter{Limit: 1})
	if len(runs) != 1 {
		t.Errorf("limit = %+v", runs)
	}
}

func TestCIJobs(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/actions/runs/103/jobs" {
			http.NotFound(w, r)
			return
		}
		w.Write(fixture(t, "jobs.json"))
	})
	jobs, err := c.CIJobs(context.Background(), 103)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].ID != 201 || jobs[0].Attempt != 2 || jobs[0].Status != "failure" || !jobs[1].NoLog {
		t.Errorf("jobs = %+v", jobs)
	}
	if _, err := c.CIJobs(context.Background(), 1); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestCILogFollowsRedirectWithoutToken(t *testing.T) {
	var leaked string
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked = r.Header.Get("Authorization")
		w.Write(fixture(t, "log.txt"))
	}))
	t.Cleanup(storage.Close)
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/actions/jobs/201/logs":
			http.Redirect(w, r, storage.URL+"/blob", http.StatusFound)
		case "/repos/o/r/actions/jobs/9/logs":
			w.WriteHeader(http.StatusGone)
		default:
			http.NotFound(w, r)
		}
	})
	log, err := c.CILog(context.Background(), 201)
	if err != nil || !strings.Contains(log, "FAIL TestX") {
		t.Fatalf("log = %q, %v", log, err)
	}
	// The two servers share the host name 127.0.0.1 but not the port, so Go counts them as two hosts.
	if leaked != "" {
		t.Errorf("token reached the storage host: %q", leaked)
	}
	if _, err := c.CILog(context.Background(), 9); !errors.Is(err, forge.ErrNoLog) {
		t.Errorf("want ErrNoLog, got %v", err)
	}
}

func TestTokenFromEnv(t *testing.T) {
	env := map[string]string{"GH_TOKEN": "b"}
	get := func(k string) string { return env[k] }
	if got := TokenFromEnv(get); got != "b" {
		t.Errorf("got %q", got)
	}
	env["GITHUB_TOKEN"] = "a"
	if got := TokenFromEnv(get); got != "a" {
		t.Errorf("got %q", got)
	}
}
