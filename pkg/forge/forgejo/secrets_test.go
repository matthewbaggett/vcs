package forgejo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSecretList(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/actions/secrets" {
			t.Errorf("unexpected %s", r.URL)
		}
		w.Write([]byte(`[{"name":"B","created_at":"t2"},{"name":"A","created_at":"t1"}]`))
	})
	ss, err := c.SecretList(context.Background())
	if err != nil || len(ss) != 2 || ss[0].Name != "A" || ss[1].Created != "t2" {
		t.Fatalf("secrets: %+v %v", ss, err)
	}
	bad := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	if _, err := bad.SecretList(context.Background()); err == nil {
		t.Errorf("a non-list body gave no error")
	}
}

func TestSecretSetAndDelete(t *testing.T) {
	var method, path, body string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(204)
	})
	if err := c.SecretSet(context.Background(), "KEY", "val ue"); err != nil {
		t.Fatal(err)
	}
	if method != "PUT" || path != "/api/v1/repos/o/r/actions/secrets/KEY" || body != `{"data":"val ue"}` {
		t.Errorf("set: %s %s %s", method, path, body)
	}
	if err := c.SecretDelete(context.Background(), "KEY"); err != nil || method != "DELETE" || path != "/api/v1/repos/o/r/actions/secrets/KEY" {
		t.Errorf("delete: %s %s %v", method, path, err)
	}
	fail := newServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "nope", 500) })
	if err := fail.SecretSet(context.Background(), "KEY", "topsecret"); err == nil || strings.Contains(err.Error(), "topsecret") {
		t.Errorf("error holds the value or is nil: %v", err)
	}
}
