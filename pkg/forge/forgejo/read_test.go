package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func TestPRList(t *testing.T) {
	var query string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/pulls" {
			http.NotFound(w, r)
			return
		}
		query = r.URL.RawQuery
		w.Write([]byte(`[
		 {"number":5,"title":"Five","mergeable":true,"head":{"ref":"feature/a"},"base":{"ref":"main"},"labels":[{"name":"x"},{"name":"y"}],"milestone":{"title":"v1"}},
		 {"number":4,"title":"Four","mergeable":false,"head":{"ref":"fix/b"},"base":{"ref":"main"},"labels":[],"milestone":null}
		]`))
	})
	got, err := c.PRList(context.Background(), forge.ListFilter{})
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v err %v", got, err)
	}
	if !strings.Contains(query, "state=open") || !strings.Contains(query, "limit=50") || !strings.Contains(query, "page=1") {
		t.Errorf("query = %q", query)
	}
	if p := got[0]; p.Number != 5 || p.Head != "feature/a" || p.Base != "main" || !p.Mergeable || len(p.Labels) != 2 || p.Milestone != "v1" {
		t.Errorf("row = %+v", p)
	}
	// The filter runs on our side: labels must all match, and the milestone must match.
	got, _ = c.PRList(context.Background(), forge.ListFilter{State: "all", Labels: []string{"x", "y"}, Milestone: "v1"})
	if len(got) != 1 || got[0].Number != 5 || !strings.Contains(query, "state=all") {
		t.Errorf("filtered = %+v, query %q", got, query)
	}
	got, _ = c.PRList(context.Background(), forge.ListFilter{Limit: 1})
	if len(got) != 1 || got[0].Number != 5 {
		t.Errorf("limit = %+v", got)
	}
}

func TestIssueListReadsEveryPage(t *testing.T) {
	var pages []string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/issues" || r.URL.Query().Get("type") != "issues" {
			http.NotFound(w, r)
			return
		}
		pages = append(pages, r.URL.Query().Get("page"))
		if r.URL.Query().Get("page") == "1" {
			var rows []string
			for i := 0; i < 50; i++ {
				rows = append(rows, fmt.Sprintf(`{"number":%d,"title":"t","state":"open"}`, 100-i))
			}
			w.Write([]byte("[" + strings.Join(rows, ",") + "]"))
			return
		}
		w.Write([]byte(`[{"number":9,"title":"last","state":"open"}]`))
	})
	got, err := c.IssueList(context.Background(), forge.ListFilter{})
	if err != nil || len(got) != 51 || got[50].Number != 9 {
		t.Fatalf("len %d err %v", len(got), err)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Errorf("pages = %v", pages)
	}
}

func TestIssueListSendsTheQuery(t *testing.T) {
	var q string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query().Get("q")
		w.Write([]byte(`[{"number":3,"title":"hit","state":"open"}]`))
	})
	got, err := c.IssueList(context.Background(), forge.ListFilter{Query: "retry loop"})
	if err != nil || len(got) != 1 || q != "retry loop" {
		t.Fatalf("got %v, err %v, q %q", got, err, q)
	}
	if _, err := c.IssueList(context.Background(), forge.ListFilter{}); err != nil || q != "" {
		t.Errorf("an empty query must send no q, got %q", q)
	}
}

func TestIssueView(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/o/r/issues/8":
			w.Write([]byte(`{"number":8,"title":"Thin","state":"open","body":"Body","labels":[{"name":"plan"}],"milestone":{"title":"v1"}}`))
		case "/api/v1/repos/o/r/issues/8/comments":
			w.Write([]byte(`[{"user":{"login":"ann"},"created_at":"2026-01-02T00:00:00Z","body":"Hi"}]`))
		default:
			http.NotFound(w, r)
		}
	})
	got, err := c.IssueView(context.Background(), 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 8 || got.Title != "Thin" || got.Body != "Body" || got.Milestone != "v1" || len(got.Comments) != 1 || got.Comments[0].Author != "ann" {
		t.Errorf("issue = %+v", got)
	}
	if _, err := c.IssueView(context.Background(), 99); err == nil {
		t.Error("want an error for a missing issue")
	}
}
