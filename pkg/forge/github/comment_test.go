package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCommentManager(t *testing.T) {
	var sent []string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = append(sent, r.Method+" "+r.URL.Path+" "+string(b))
		if r.Method == http.MethodGet {
			w.Write([]byte(`[{"id":501,"user":{"login":"maria"},"created_at":"c1","updated_at":"u1","body":"Hi"}]`))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	rows, err := c.CommentList(ctx, 14)
	if err != nil || len(rows) != 1 || rows[0].ID != 501 || rows[0].Author != "maria" || rows[0].Updated != "u1" {
		t.Fatalf("list: %+v %v", rows, err)
	}
	if err := c.CommentEdit(ctx, 501, "New"); err != nil {
		t.Fatal(err)
	}
	if err := c.CommentDelete(ctx, 501); err != nil {
		t.Fatal(err)
	}
	if sent[0] != "GET /repos/o/r/issues/14/comments " {
		t.Errorf("list sent %q", sent[0])
	}
	if sent[1] != "PATCH /repos/o/r/issues/comments/501 {\"body\":\"New\"}" {
		t.Errorf("edit sent %q", sent[1])
	}
	if !strings.HasPrefix(sent[2], "DELETE /repos/o/r/issues/comments/501") {
		t.Errorf("delete sent %q", sent[2])
	}
}
