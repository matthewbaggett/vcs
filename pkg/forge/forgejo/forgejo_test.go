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

func TestPRWatch(t *testing.T) {
	page := func(rows string) string { return rows }
	var protection = `{"status_check_contexts":["CI / * (pull_request)"]}`
	var rule int
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/o/r/pulls/7":
			w.Write([]byte(`{"state":"open","merged":false,"head":{"sha":"abc1234def"},"base":{"ref":"main"}}`))
		case "/api/v1/repos/o/r/pulls/8":
			w.Write([]byte(`{"state":"closed","merged":true,"head":{"sha":"abc1234def"},"base":{"ref":"main"}}`))
		case "/api/v1/repos/o/r/commits/abc1234def/statuses":
			if r.URL.Query().Get("page") == "2" {
				w.Write([]byte(page(`[{"id":9,"context":"CI / z (pull_request)","status":"failure","target_url":"/run/9"}]`)))
				return
			}
			// 50 rows, so that the driver asks for page 2. The newest row of CI / a wins.
			var rows []string
			rows = append(rows, `{"id":1,"context":"CI / a (pull_request)","status":"failure","target_url":"/run/1"}`)
			rows = append(rows, `{"id":2,"context":"CI / a (pull_request)","status":"success","target_url":"/run/2"}`)
			for i := 3; i <= 50; i++ {
				rows = append(rows, `{"id":`+strings.Repeat("1", 1)+`00`+string(rune('0'+i%10))+`,"context":"CI / p`+string(rune('a'+i%26))+string(rune('a'+i/26))+` (pull_request)","status":"success","target_url":""}`)
			}
			w.Write([]byte("[" + strings.Join(rows, ",") + "]"))
		case "/api/v1/repos/o/r/branch_protections/main":
			rule++
			if rule > 1 {
				http.Error(w, "no", http.StatusForbidden)
				return
			}
			w.Write([]byte(protection))
		default:
			http.NotFound(w, r)
		}
	})
	w, err := c.PRWatch(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if w.State != "open" || w.HeadSHA != "abc1234def" || w.Base != "main" {
		t.Errorf("state = %+v", w)
	}
	if len(w.Required) != 1 || w.Required[0] != "CI / * (pull_request)" {
		t.Errorf("required = %v", w.Required)
	}
	got := map[string]forge.Check{}
	for _, ck := range w.Checks {
		got[ck.Context] = ck
	}
	if a := got["CI / a (pull_request)"]; a.Status != "success" || a.URL != "/run/2" {
		t.Errorf("newest row of CI / a: %+v", a)
	}
	if z := got["CI / z (pull_request)"]; z.Status != "failure" || z.URL != "/run/9" {
		t.Errorf("a status on page 2 is missing: %+v", z)
	}
	// A rule that the token can't read is no rule.
	w, err = c.PRWatch(context.Background(), 7)
	if err != nil || len(w.Required) != 0 {
		t.Errorf("unreadable rule: %v %v", w.Required, err)
	}
	// A merged pull request reads no statuses.
	w, err = c.PRWatch(context.Background(), 8)
	if err != nil || w.State != "merged" || len(w.Checks) != 0 {
		t.Errorf("merged: %+v %v", w, err)
	}
	if _, err := c.PRWatch(context.Background(), 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestCommitStatuses(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/commits/abc/statuses" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("page") != "1" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(`[{"id":9,"context":"ci / test","status":"success","target_url":"/o/r/actions/runs/1","description":"ok"},
 {"id":7,"context":"ci / lint","status":"failure","target_url":"https://x/y","description":""}]`))
	})
	rows, err := c.CommitStatuses(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != 9 || rows[0].Description != "ok" || rows[1].URL != "https://x/y" {
		t.Fatalf("rows = %+v", rows)
	}
	if !strings.HasSuffix(rows[0].URL, "/o/r/actions/runs/1") || !strings.HasPrefix(rows[0].URL, "http") {
		t.Errorf("path address not completed: %q", rows[0].URL)
	}
	if _, err := c.CommitStatuses(context.Background(), "nope"); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestIssueTimeline(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/issues/5/timeline" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("page") != "1" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(`[
 {"id":4,"created_at":"2026-01-02T00:00:00Z","type":"pull_push","body":"{\"is_force_push\":true,\"commit_ids\":[\"aaa\",\"bbb\"]}","user":{"login":"u"}},
 {"id":3,"created_at":"2026-01-01T00:00:00Z","type":"change_title","old_title":"a","new_title":"b","user":{"login":"u"}},
 {"id":2,"created_at":"2026-01-01T00:00:00Z","type":"label","body":"1","label":{"name":"x"},"user":{"login":"u"}},
 {"id":1,"created_at":"2026-01-01T00:00:00Z","type":"label","body":"","label":{"name":"y"},"user":{"login":"u"}},
 {"id":5,"created_at":"2026-01-03T00:00:00Z","type":"commit_ref","ref_commit_sha":"abc","body":"x","user":{"login":"u"}}]`))
	})
	ev, err := c.IssueTimeline(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range ev {
		got = append(got, e.Detail)
	}
	want := []string{"-y", "+x", "a -> b", "force-push aaa,bbb", "abc"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("details = %q, want %q", got, want)
	}
	if _, err := c.IssueTimeline(context.Background(), 6); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
