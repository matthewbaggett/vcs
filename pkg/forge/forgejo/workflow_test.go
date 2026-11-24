package forgejo

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestWorkflowDispatch(t *testing.T) {
	var method, path, body string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(204)
	})
	if err := c.WorkflowDispatch(context.Background(), "nightly.yml", "main"); err != nil {
		t.Fatal(err)
	}
	if method != "POST" || path != "/api/v1/repos/o/r/actions/workflows/nightly.yml/dispatches" || body != `{"ref":"main"}` {
		t.Errorf("dispatch: %s %s %s", method, path, body)
	}
	bad := newServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 404) })
	if err := bad.WorkflowDispatch(context.Background(), "x.yml", "main"); err == nil {
		t.Errorf("a 404 gave no error")
	}
}
