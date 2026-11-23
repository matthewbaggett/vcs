package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type recorder struct {
	mu    sync.Mutex
	posts []string
}

func (r *recorder) add(s string) { r.mu.Lock(); r.posts = append(r.posts, s); r.mu.Unlock() }

func rerunServer(t *testing.T, rec *recorder, loggedIn bool) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/o/r/actions/runs":
			if r.URL.Query().Get("page") == "1" {
				w.Write(fixture(t, "runs1.json"))
			} else {
				w.Write([]byte(`{"workflow_runs":[]}`))
			}
		case strings.HasSuffix(r.URL.Path, "/actions/runs/103/jobs"):
			w.Write([]byte(`[{"id":1,"name":"a","status":"success","runs_on":["x"]},{"id":2,"name":"b","status":"failure","runs_on":["x"]},{"id":3,"name":"c","status":"failure","runs_on":["x"]}]`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rerun"):
			if c, err := r.Cookie("i_like_gitea"); err != nil || c.Value != "good" || !loggedIn {
				w.Header().Set("Location", "/user/login")
				w.WriteHeader(http.StatusSeeOther)
				return
			}
			if r.Header.Get("Origin") == "" || r.Header.Get("Sec-Fetch-Site") != "same-origin" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			rec.add(r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok", Cookie: "good"})
}

func TestRerunWholeRun(t *testing.T) {
	rec := &recorder{}
	c := rerunServer(t, rec, true)
	if err := c.CIRerun(context.Background(), 103, false); err != nil {
		t.Fatal(err)
	}
	if len(rec.posts) != 1 || rec.posts[0] != "/o/r/actions/runs/30/rerun" {
		t.Errorf("posts = %v", rec.posts)
	}
}

func TestRerunFailedJobsOnly(t *testing.T) {
	rec := &recorder{}
	c := rerunServer(t, rec, true)
	if err := c.CIRerun(context.Background(), 103, true); err != nil {
		t.Fatal(err)
	}
	want := []string{"/o/r/actions/runs/30/jobs/1/rerun", "/o/r/actions/runs/30/jobs/2/rerun"}
	if strings.Join(rec.posts, ",") != strings.Join(want, ",") {
		t.Errorf("posts = %v", rec.posts)
	}
}

func TestRerunNeedsSession(t *testing.T) {
	rec := &recorder{}
	c := rerunServer(t, rec, false)
	if err := c.CIRerun(context.Background(), 103, false); !errors.Is(err, forge.ErrNoSession) {
		t.Errorf("rejected cookie: %v", err)
	}
	c.cfg.Cookie = ""
	if err := c.CIRerun(context.Background(), 103, false); !errors.Is(err, forge.ErrNoSession) {
		t.Errorf("no cookie: %v", err)
	}
	if err := c.CIRerun(context.Background(), 999, false); !errors.Is(err, forge.ErrNotFound) && !errors.Is(err, forge.ErrNoSession) {
		t.Errorf("unknown run: %v", err)
	}
}

func TestAttachAndComment(t *testing.T) {
	var gotName, gotBody, gotAuth, commentBody string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case strings.HasSuffix(r.URL.Path, "/issues/7/assets"):
			gotName = r.URL.Query().Get("name")
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
				return
			}
			f, _, _ := r.FormFile("attachment")
			b, _ := io.ReadAll(f)
			gotBody = string(b)
			w.Write([]byte(`{"browser_download_url":"https://git.example/attachments/abc"}`))
		case strings.HasSuffix(r.URL.Path, "/issues/7/comments"):
			b, _ := io.ReadAll(r.Body)
			commentBody = string(b)
			w.WriteHeader(http.StatusCreated)
		}
	})
	file := filepath.Join(t.TempDir(), "shot one.png")
	os.WriteFile(file, []byte("PNGDATA"), 0o600)
	assets, err := c.Attach(context.Background(), 7, []string{file})
	if err != nil || len(assets) != 1 || assets[0].URL != "https://git.example/attachments/abc" {
		t.Fatalf("assets=%+v err=%v", assets, err)
	}
	if gotName != "shot one.png" || gotBody != "PNGDATA" || gotAuth != "token tok" {
		t.Errorf("name=%q body=%q auth=%q", gotName, gotBody, gotAuth)
	}
	if err := c.Comment(context.Background(), 7, "hi"); err != nil || commentBody != `{"body":"hi"}` {
		t.Errorf("comment: %q %v", commentBody, err)
	}
	if _, err := c.Attach(context.Background(), 7, []string{"/no/such/file"}); err == nil {
		t.Error("missing file should fail")
	}
}

func TestCreatePR(t *testing.T) {
	var got map[string]string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/pulls") {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &got)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"number":9,"html_url":"https://git.example/o/r/pulls/9"}`))
	})
	n, addr, err := c.CreatePR(context.Background(), forge.NewPR{Title: "Add it", Body: "b", Head: "feature/x", Base: "main", Draft: true})
	if err != nil || n != 9 || addr != "https://git.example/o/r/pulls/9" {
		t.Fatalf("n=%d addr=%q err=%v", n, addr, err)
	}
	if got["title"] != "WIP: Add it" || got["head"] != "feature/x" || got["base"] != "main" {
		t.Errorf("payload %v", got)
	}
}
