package forgejo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	return New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok", RetrySleep: time.Millisecond})
}

func TestPRStatus(t *testing.T) {
	var auth string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/api/v1/repos/o/r/pulls/12":
			w.Write(fixture(t, "pr12.json"))
		case "/api/v1/repos/o/r/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/statuses":
			w.Write(fixture(t, "statuses.json"))
		default:
			http.NotFound(w, r)
		}
	})
	s, err := c.PRStatus(context.Background(), 12)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "token tok" {
		t.Errorf("auth header = %q", auth)
	}
	// ci / test: newest row is success. ci / lint is pending. The worst is pending.
	if s.State != "open" || !s.Draft || s.Mergeable || s.CI != "pending" || s.URL == "" {
		t.Errorf("status = %+v", s)
	}
	if _, err := c.PRStatus(context.Background(), 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestSummarize(t *testing.T) {
	cases := map[string][]string{
		"none":    nil,
		"failure": {"success", "error", "pending"},
		"pending": {"success", "pending", "warning"},
		"warning": {"success", "warning"},
		"success": {"success"},
	}
	for want, in := range cases {
		if got := Summarize(in); got != want {
			t.Errorf("Summarize(%v) = %s, want %s", in, got, want)
		}
	}
}

func TestCIRunsFiltersOnThisSide(t *testing.T) {
	var pages int32
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "" {
			t.Error("request without page")
		}
		atomic.AddInt32(&pages, 1)
		if r.URL.Query().Get("page") == "1" {
			w.Write(fixture(t, "runs1.json"))
			return
		}
		w.Write([]byte(`{"workflow_runs":[]}`))
	})
	all, err := c.CIRuns(context.Background(), forge.RunFilter{Limit: 20})
	if err != nil || len(all) != 3 || all[0].ID != 103 {
		t.Fatalf("all = %+v, %v", all, err)
	}
	main, _ := c.CIRuns(context.Background(), forge.RunFilter{Branch: "main", Limit: 1})
	if len(main) != 1 || main[0].ID != 102 {
		t.Errorf("main = %+v", main)
	}
	pr, _ := c.CIRuns(context.Background(), forge.RunFilter{Branch: "#12"})
	if len(pr) != 1 || pr[0].Status != "failure" {
		t.Errorf("pr = %+v", pr)
	}
	sha, _ := c.CIRuns(context.Background(), forge.RunFilter{SHA: "bbbb"})
	if len(sha) != 1 || sha[0].ID != 102 {
		t.Errorf("sha = %+v", sha)
	}
	st, _ := c.CIRuns(context.Background(), forge.RunFilter{Status: "success"})
	if len(st) != 2 {
		t.Errorf("status = %+v", st)
	}
}

func TestCIJobsBothShapes(t *testing.T) {
	for name, wantFirst := range map[string]int64{"jobs_array.json": 501, "jobs_wrapped.json": 601} {
		c := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write(fixture(t, name)) })
		jobs, err := c.CIJobs(context.Background(), 7)
		if err != nil || len(jobs) == 0 || jobs[0].ID != wantFirst {
			t.Fatalf("%s: %+v, %v", name, jobs, err)
		}
		if name == "jobs_array.json" && (!jobs[1].NoLog || jobs[1].Attempt != 1 || jobs[0].Attempt != 2) {
			t.Errorf("jobs = %+v", jobs)
		}
	}
}

func TestCILogRetriesOn504(t *testing.T) {
	var n int32
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) < 3 {
			http.Error(w, "busy", http.StatusGatewayTimeout)
			return
		}
		w.Write(fixture(t, "log.txt"))
	})
	text, err := c.CILog(context.Background(), 501)
	if err != nil || !strings.Contains(text, "TestX") || n != 3 {
		t.Fatalf("text=%q err=%v tries=%d", text, err, n)
	}
}

func TestCILogGivesUpAfterThreeTries(t *testing.T) {
	var n int32
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		http.Error(w, "busy", http.StatusGatewayTimeout)
	})
	_, err := c.CILog(context.Background(), 501)
	var fe *forge.Error
	if !errors.As(err, &fe) || fe.Status != 504 || n != 3 {
		t.Fatalf("err=%v tries=%d", err, n)
	}
}

func TestCILogNoLogFile(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "open logs for task 5: file does not exist", http.StatusInternalServerError)
	})
	if _, err := c.CILog(context.Background(), 502); !errors.Is(err, forge.ErrNoLog) {
		t.Fatalf("want ErrNoLog, got %v", err)
	}
}

func TestBasicAuthFallback(t *testing.T) {
	var user, pass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ = r.BasicAuth()
		w.Write([]byte(`{"workflow_runs":[]}`))
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, Repo: "o/r", Login: "me", Password: "pw"})
	if _, err := c.CIRuns(context.Background(), forge.RunFilter{}); err != nil || user != "me" || pass != "pw" {
		t.Fatalf("user=%q pass=%q err=%v", user, pass, err)
	}
}

func TestCommitTime(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/git/commits/abc" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"commit":{"committer":{"date":"2026-10-17T12:00:00+02:00"}}}`))
	})
	got, err := c.CommitTime(context.Background(), "abc")
	if err != nil || got != "2026-10-17T10:00:00Z" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := c.CommitTime(context.Background(), "missing"); err == nil {
		t.Fatal("a missing commit must fail")
	}
}
