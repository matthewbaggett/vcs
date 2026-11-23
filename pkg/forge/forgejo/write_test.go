package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestCreateIssueAndReady(t *testing.T) {
	var patched, issue map[string]string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
			json.Unmarshal(b, &issue)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"number":4,"html_url":"https://git.example/o/r/issues/4"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pulls/3"):
			w.Write([]byte(`{"title":"WIP: Add it"}`))
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/pulls/3"):
			json.Unmarshal(b, &patched)
			w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	n, addr, err := c.CreateIssue(context.Background(), "T", "B")
	if err != nil || n != 4 || addr == "" || issue["title"] != "T" || issue["body"] != "B" {
		t.Fatalf("issue: n=%d addr=%q err=%v got=%v", n, addr, err, issue)
	}
	if err := c.PRReady(context.Background(), 3); err != nil || patched["title"] != "Add it" {
		t.Fatalf("ready: err=%v patched=%v", err, patched)
	}
}

func TestPREditKeepsDraft(t *testing.T) {
	var patched map[string]string
	title := "WIP: Old"
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"title":"` + title + `"}`))
		case http.MethodPatch:
			patched = nil
			json.Unmarshal(b, &patched)
			w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	ctx := context.Background()
	nt, nb := "New", "New body"
	if err := c.PREdit(ctx, 3, forge.PREdit{Title: &nt}); err != nil || patched["title"] != "WIP: New" || len(patched) != 1 {
		t.Fatalf("draft title: err=%v patched=%v", err, patched)
	}
	title = "Old"
	if err := c.PREdit(ctx, 3, forge.PREdit{Title: &nt, Body: &nb}); err != nil || patched["title"] != "New" || patched["body"] != "New body" {
		t.Fatalf("ready title and body: err=%v patched=%v", err, patched)
	}
	patched = map[string]string{"x": "y"}
	if err := c.PREdit(ctx, 3, forge.PREdit{}); err != nil || patched["x"] != "y" {
		t.Fatalf("empty edit sent a call: err=%v patched=%v", err, patched)
	}
}

func TestLabels(t *testing.T) {
	var sent []string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = append(sent, r.Method+" "+r.URL.Path+" "+string(b))
		if r.Method == http.MethodGet {
			w.Write([]byte(`[{"id":4,"name":"bug","color":"d73a4a","description":"Broken"}]`))
			return
		}
		w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	ls, err := c.LabelList(ctx)
	if err != nil || len(ls) != 1 || ls[0].ID != 4 || ls[0].Color != "d73a4a" {
		t.Fatalf("list: %v %v", ls, err)
	}
	if err := c.LabelCreate(ctx, "wip", "aabbcc", "Work"); err != nil {
		t.Fatal(err)
	}
	if err := c.LabelRename(ctx, 4, "defect"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sent[1], `"color":"#aabbcc"`) || !strings.HasPrefix(sent[1], "POST ") {
		t.Errorf("create sent %q", sent[1])
	}
	if !strings.HasPrefix(sent[2], "PATCH ") || !strings.HasSuffix(sent[2], "/labels/4 {\"name\":\"defect\"}") {
		t.Errorf("rename sent %q", sent[2])
	}
}

func TestPRMerge(t *testing.T) {
	var posted map[string]any
	merged := false
	refuse := false
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls/9/merge"):
			if refuse {
				w.WriteHeader(http.StatusMethodNotAllowed)
				w.Write([]byte(`{"message":"not mergeable"}`))
				return
			}
			b, _ := io.ReadAll(r.Body)
			posted = nil
			json.Unmarshal(b, &posted)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pulls/9"):
			w.Write([]byte(fmt.Sprintf(`{"merged":%v}`, merged)))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	ctx := context.Background()
	if err := c.PRMerge(ctx, 9, forge.PRMerge{}); err != nil || posted["Do"] != "merge" || len(posted) != 1 {
		t.Fatalf("default: err=%v posted=%v", err, posted)
	}
	if err := c.PRMerge(ctx, 9, forge.PRMerge{Method: "squash", DeleteBranch: true}); err != nil || posted["Do"] != "squash" || posted["delete_branch_after_merge"] != true {
		t.Fatalf("squash: err=%v posted=%v", err, posted)
	}
	refuse = true
	if err := c.PRMerge(ctx, 9, forge.PRMerge{}); err == nil || strings.Contains(err.Error(), "merged already") {
		t.Fatalf("refused: err=%v", err)
	}
	merged = true
	if err := c.PRMerge(ctx, 9, forge.PRMerge{}); err == nil || !strings.Contains(err.Error(), "merged already") {
		t.Fatalf("merged already: err=%v", err)
	}
}

const boardPage = `<div class="project-column" data-id="5">
<span class="project-column-title-label">Doing</span>
<div class="issue-card tw-x" data-issue="90">
<a class="issue-card-title muted" href="/o/r/issues/3">A &amp; B</a>
</div>
<div class="issue-card tw-x" data-issue="91">
<a class="issue-card-title muted" href="/o/r/pulls/4">Other</a>
</div>
<div class="project-column" data-id="6">
<span class="project-column-title-label">Done</span>
`

func TestBoard(t *testing.T) {
	var moved, added string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		switch {
		case r.URL.Path == "/o/r/projects" && r.URL.Query().Get("page") == "1":
			w.Write([]byte(`<a class="muted tw-break-anywhere" href="/o/r/projects/2">Plan &amp; go</a>`))
		case r.URL.Path == "/o/r/projects":
			w.Write([]byte(`<a class="muted tw-break-anywhere" href="/o/r/projects/2">Plan &amp; go</a>`))
		case r.URL.Path == "/o/r/projects/2":
			w.Write([]byte(boardPage))
		case r.URL.Path == "/api/v1/repos/o/r/issues/3":
			w.Write([]byte(`{"id":90}`))
		case r.URL.Path == "/api/v1/repos/o/r/issues/7":
			w.Write([]byte(`{"id":97}`))
		case r.URL.Path == "/o/r/issues/projects":
			added = string(b)
			w.Write([]byte(`{"ok":true}`))
		case r.URL.Path == "/o/r/projects/2/6/move":
			moved = string(b)
			w.Write([]byte(`{"ok":true}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok", Cookie: "sess"})
	ctx := context.Background()
	bs, err := c.BoardList(ctx, false)
	if err != nil || len(bs) != 1 || bs[0].ID != "2" || bs[0].Title != "Plan & go" {
		t.Fatalf("list: %v %v", bs, err)
	}
	cs, err := c.BoardView(ctx, "2")
	if err != nil || len(cs) != 2 || cs[0].Title != "A & B" || cs[0].Column != "Doing" || cs[0].ItemID != "90" || cs[1].Number != 4 {
		t.Fatalf("view: %+v %v", cs, err)
	}
	if err := c.BoardAdd(ctx, "2", []int{3}); err != nil || !strings.Contains(added, "issue_ids=90") {
		t.Fatalf("add: %v %q", err, added)
	}
	// Done has no cards, so its id comes from the page.
	if err := c.BoardMove(ctx, "2", "done", []int{7}); err != nil || moved != `{"issues":[{"issueID":97,"sorting":0}]}` {
		t.Fatalf("move: %v %q", err, moved)
	}
	anon := New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok"})
	if err := anon.BoardAdd(ctx, "2", []int{3}); !errors.Is(err, forge.ErrNoSession) {
		t.Errorf("no session: %v", err)
	}
}
