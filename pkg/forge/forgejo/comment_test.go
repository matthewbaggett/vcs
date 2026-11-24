package forgejo

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
		w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	rows, err := c.CommentList(ctx, 14)
	if err != nil || len(rows) != 1 || rows[0].ID != 501 || rows[0].Author != "maria" || rows[0].Updated != "u1" || rows[0].Body != "Hi" {
		t.Fatalf("list: %+v %v", rows, err)
	}
	if err := c.CommentEdit(ctx, 501, "New"); err != nil {
		t.Fatal(err)
	}
	if err := c.CommentDelete(ctx, 501); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sent[0], "GET ") || !strings.Contains(sent[0], "/issues/14/comments") {
		t.Errorf("list sent %q", sent[0])
	}
	if !strings.HasPrefix(sent[1], "PATCH ") || !strings.HasSuffix(sent[1], "/issues/comments/501 {\"body\":\"New\"}") {
		t.Errorf("edit sent %q", sent[1])
	}
	if !strings.HasPrefix(sent[2], "DELETE ") || !strings.HasSuffix(sent[2], "/issues/comments/501 ") {
		t.Errorf("delete sent %q", sent[2])
	}
}
