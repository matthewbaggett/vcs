package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func TestPRDraft(t *testing.T) {
	isDraft := false
	var mutations int
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var in struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(raw, &in)
		switch {
		case strings.Contains(in.Query, "convertPullRequestToDraft"):
			mutations++
			w.Write([]byte(`{"data":{"convertPullRequestToDraft":{"pullRequest":{"id":"PR_1"}}}}`))
		default:
			d := "false"
			if isDraft {
				d = "true"
			}
			w.Write([]byte(`{"data":{"repository":{"pullRequest":{"id":"PR_1","isDraft":` + d + `}}}}`))
		}
	})
	ctx := context.Background()
	if err := c.PRDraft(ctx, 7); err != nil || mutations != 1 {
		t.Fatalf("draft: err=%v mutations=%d", err, mutations)
	}
	isDraft = true
	if err := c.PRDraft(ctx, 7); err != nil || mutations != 1 {
		t.Fatalf("draft twice: err=%v mutations=%d", err, mutations)
	}
}

func TestPRCloseAndRebase(t *testing.T) {
	var sent string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = r.Method + " " + r.URL.Path + " " + string(b)
		w.Write([]byte(`{}`))
	})
	if err := c.PRClose(context.Background(), 7); err != nil || sent != `PATCH /repos/o/r/pulls/7 {"state":"closed"}` {
		t.Fatalf("close: err=%v sent=%q", err, sent)
	}
	if _, err := c.PRRebase(context.Background(), 7); !errors.Is(err, forge.ErrUnsupported) {
		t.Fatalf("rebase: want ErrUnsupported, got %v", err)
	}
}
