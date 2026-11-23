package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type call struct {
	Method, Path, Query string
	Vars                map[string]any
	Body                map[string]any
}

type rec struct {
	mu    sync.Mutex
	calls []call
}

func (r *rec) add(c call) { r.mu.Lock(); r.calls = append(r.calls, c); r.mu.Unlock() }

func (r *rec) find(sub string) []call {
	var out []call
	for _, c := range r.calls {
		if strings.Contains(c.Query, sub) || strings.Contains(c.Path, sub) {
			out = append(out, c)
		}
	}
	return out
}

// recorded answers like GitHub does, from canned JSON, and keeps each call.
func recorded(t *testing.T) (*Client, *rec) {
	t.Helper()
	r := &rec{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		c := call{Method: req.Method, Path: req.URL.Path}
		var in struct {
			Query string         `json:"query"`
			Vars  map[string]any `json:"variables"`
		}
		if json.Unmarshal(raw, &in) == nil {
			c.Query, c.Vars = in.Query, in.Vars
			_ = json.Unmarshal(raw, &c.Body)
		}
		r.add(c)
		switch {
		case req.URL.Path == "/repos/o/r/pulls" && req.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"number":7,"html_url":"https://github.com/o/r/pull/7"}`))
		case req.URL.Path == "/repos/o/r/issues" && req.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"number":8,"html_url":"https://github.com/o/r/issues/8"}`))
		case strings.HasSuffix(req.URL.Path, "/comments"), strings.Contains(req.URL.Path, "/rerun"):
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{}`))
		case req.URL.Path != "/graphql":
			http.NotFound(w, req)
		case strings.Contains(in.Query, "projectsV2("):
			w.Write(fixture(t, "gql_projects.json"))
		case strings.Contains(in.Query, "projectV2(number"):
			w.Write([]byte(`{"data":{"repository":{"projectV2":{"id":"PVT_1"}}}}`))
		case strings.Contains(in.Query, "items(first"):
			w.Write(fixture(t, "gql_items.json"))
		case strings.Contains(in.Query, "ProjectV2SingleSelectField"):
			w.Write(fixture(t, "gql_field.json"))
		case strings.Contains(in.Query, "issueOrPullRequest"):
			w.Write([]byte(`{"data":{"repository":{"issueOrPullRequest":{"id":"I_` + strings.TrimSpace(jsonNum(in.Vars["n"])) + `"}}}}`))
		case strings.Contains(in.Query, "addProjectV2ItemById"):
			w.Write([]byte(`{"data":{"addProjectV2ItemById":{"item":{"id":"ITEM_NEW"}}}}`))
		case strings.Contains(in.Query, "updateProjectV2ItemFieldValue"):
			w.Write([]byte(`{"data":{"updateProjectV2ItemFieldValue":{"projectV2Item":{"id":"x"}}}}`))
		case strings.Contains(in.Query, "pullRequest(number"):
			if jsonNum(in.Vars["n"]) == "99" {
				w.Write([]byte(`{"data":{"repository":{"pullRequest":null}}}`))
				return
			}
			w.Write([]byte(`{"data":{"repository":{"pullRequest":{"id":"PR_1","isDraft":true}}}}`))
		case strings.Contains(in.Query, "markPullRequestReadyForReview"):
			w.Write([]byte(`{"data":{"markPullRequestReadyForReview":{"pullRequest":{"id":"PR_1"}}}}`))
		default:
			w.Write([]byte(`{"errors":[{"type":"NOT_FOUND","message":"no route in the test"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return New(Config{BaseURL: srv.URL, Repo: "o/r", Token: "tok"}), r
}

func jsonNum(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestBoardList(t *testing.T) {
	c, _ := recorded(t)
	open, err := c.BoardList(context.Background(), false)
	if err != nil || len(open) != 1 || open[0].ID != "PVT_1" || open[0].Title != "Roadmap" {
		t.Fatalf("open = %+v, %v", open, err)
	}
	closed, _ := c.BoardList(context.Background(), true)
	if len(closed) != 1 || closed[0].ID != "PVT_2" {
		t.Errorf("closed = %+v", closed)
	}
}

func TestBoardView(t *testing.T) {
	c, r := recorded(t)
	cards, err := c.BoardView(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	// The draft card has no number, so it gives no row.
	if len(cards) != 2 || cards[0].Number != 3 || cards[0].Column != "Todo" || cards[0].ColumnID != "opt_todo" || cards[0].ItemID != "ITEM_3" {
		t.Errorf("cards = %+v", cards)
	}
	if cards[1].Number != 4 || cards[1].Column != "" {
		t.Errorf("card without status = %+v", cards[1])
	}
	if len(r.find("projectV2(number")) != 1 {
		t.Errorf("the number was not resolved")
	}
}

func TestBoardAdd(t *testing.T) {
	c, r := recorded(t)
	if err := c.BoardAdd(context.Background(), "PVT_1", []int{5, 6}); err != nil {
		t.Fatal(err)
	}
	adds := r.find("addProjectV2ItemById")
	if len(adds) != 2 || adds[0].Vars["p"] != "PVT_1" || adds[0].Vars["c"] != "I_5" {
		t.Errorf("adds = %+v", adds)
	}
}

func TestBoardMove(t *testing.T) {
	c, r := recorded(t)
	// #3 is on the board. #9 is not, so it is added first. The column comes by title.
	if err := c.BoardMove(context.Background(), "PVT_1", "done", []int{3, 9}); err != nil {
		t.Fatal(err)
	}
	ups := r.find("updateProjectV2ItemFieldValue")
	if len(ups) != 2 || ups[0].Vars["i"] != "ITEM_3" || ups[1].Vars["i"] != "ITEM_NEW" || ups[0].Vars["o"] != "opt_done" || ups[0].Vars["f"] != "FIELD_1" {
		t.Errorf("updates = %+v", ups)
	}
	if len(r.find("addProjectV2ItemById")) != 1 {
		t.Errorf("want one add")
	}
	// A column by id works. An unknown column is ErrNotFound.
	if err := c.BoardMove(context.Background(), "PVT_1", "opt_todo", []int{3}); err != nil {
		t.Errorf("by id: %v", err)
	}
	if err := c.BoardMove(context.Background(), "PVT_1", "Nope", []int{3}); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestCreatePR(t *testing.T) {
	c, r := recorded(t)
	n, url, err := c.CreatePR(context.Background(), forge.NewPR{Title: "T", Body: "B", Head: "h", Base: "main", Draft: true})
	if err != nil || n != 7 || !strings.HasSuffix(url, "/pull/7") {
		t.Fatalf("%d %s %v", n, url, err)
	}
	b := r.find("/repos/o/r/pulls")[0].Body
	if b["draft"] != true || b["head"] != "h" || b["base"] != "main" || b["title"] != "T" {
		t.Errorf("body = %+v", b)
	}
}

func TestCreateIssueCommentRerun(t *testing.T) {
	c, r := recorded(t)
	ctx := context.Background()
	if n, url, err := c.CreateIssue(ctx, "T", "B"); err != nil || n != 8 || !strings.HasSuffix(url, "/issues/8") {
		t.Fatalf("%d %s %v", n, url, err)
	}
	if err := c.Comment(ctx, 8, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := c.CIRerun(ctx, 5, true); err != nil {
		t.Fatal(err)
	}
	if err := c.CIRerun(ctx, 5, false); err != nil {
		t.Fatal(err)
	}
	if len(r.find("/issues/8/comments")) != 1 || len(r.find("/actions/runs/5/rerun-failed-jobs")) != 1 || len(r.find("/actions/runs/5/rerun")) != 2 {
		t.Errorf("calls = %+v", r.calls)
	}
	if _, err := c.Attach(ctx, 8, []string{"x"}); !errors.Is(err, forge.ErrUnsupported) {
		t.Errorf("attach: %v", err)
	}
}

func TestPRReady(t *testing.T) {
	c, r := recorded(t)
	if err := c.PRReady(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	m := r.find("markPullRequestReadyForReview")
	if len(m) != 1 || m[0].Vars["id"] != "PR_1" {
		t.Errorf("mutation = %+v", m)
	}
	if err := c.PRReady(context.Background(), 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
