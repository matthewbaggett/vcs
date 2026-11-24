package forgejo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPRDraftAndClose(t *testing.T) {
	title := "Add it"
	var sent []string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodGet {
			sent = append(sent, r.Method+" "+string(b))
		}
		w.Write([]byte(`{"title":"` + title + `","body":""}`))
	})
	ctx := context.Background()
	if err := c.PRDraft(ctx, 5); err != nil || len(sent) != 1 || sent[0] != `PATCH {"title":"WIP: Add it"}` {
		t.Fatalf("draft: err=%v sent=%v", err, sent)
	}
	title = "WIP: Add it"
	sent = nil
	if err := c.PRDraft(ctx, 5); err != nil || len(sent) != 0 {
		t.Fatalf("draft twice: err=%v sent=%v", err, sent)
	}
	if err := c.PRClose(ctx, 5); err != nil || len(sent) != 1 || sent[0] != `PATCH {"state":"closed"}` {
		t.Fatalf("close: err=%v sent=%v", err, sent)
	}
}

func TestPRRebase(t *testing.T) {
	body := "Text\\n\\n- [ ] <!-- rebase-check -->If you want to rebase"
	var sent []string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodGet {
			sent = append(sent, string(b))
		}
		w.Write([]byte(`{"title":"t","body":"` + body + `"}`))
	})
	ctx := context.Background()
	ticked, err := c.PRRebase(ctx, 21)
	if err != nil || !ticked || len(sent) != 1 {
		t.Fatalf("tick: ticked=%v err=%v sent=%v", ticked, err, sent)
	}
	var got struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal([]byte(sent[0]), &got); err != nil || !strings.Contains(got.Body, "- [x] <!-- rebase-check -->") {
		t.Fatalf("tick body: err=%v body=%q", err, got.Body)
	}
	body = "- [x] <!-- rebase-check -->done"
	sent = nil
	if ticked, err = c.PRRebase(ctx, 21); err != nil || ticked || len(sent) != 0 {
		t.Fatalf("already: ticked=%v err=%v sent=%v", ticked, err, sent)
	}
	body = "No box here"
	if _, err = c.PRRebase(ctx, 21); err == nil || !strings.Contains(err.Error(), "no rebase box") {
		t.Fatalf("no box: err=%v", err)
	}
}
