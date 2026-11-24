package forgejo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPackageList(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packages/o" || r.URL.Query().Get("type") != "generic" || r.URL.Query().Get("q") != "tool" {
			t.Errorf("unexpected %s", r.URL)
		}
		w.Write([]byte(`[{"type":"generic","name":"tool","version":"1","created_at":"2026-01-01T00:00:00Z"},{"type":"generic","name":"tool-x","version":"2","created_at":"t"},{"type":"npm","name":"tool","version":"3","created_at":"t"}]`))
	})
	ps, err := c.PackageList(context.Background(), "tool")
	if err != nil || len(ps) != 1 || ps[0].Version != "1" {
		t.Fatalf("packages: %+v %v", ps, err)
	}
	all := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("q") {
			t.Errorf("an empty name sent q: %s", r.URL)
		}
		w.Write([]byte(`[{"type":"generic","name":"a","version":"1"},{"type":"generic","name":"b","version":"1"}]`))
	})
	if ps, err := all.PackageList(context.Background(), ""); err != nil || len(ps) != 2 {
		t.Errorf("all: %+v %v", ps, err)
	}
	bad := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	if _, err := bad.PackageList(context.Background(), ""); err == nil {
		t.Errorf("a non-list body gave no error")
	}
}

func TestPackageWrites(t *testing.T) {
	var method, path, body, ctype string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, ctype = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(201)
	})
	if err := c.PackageUpload(context.Background(), "tool", "1.0", "tool.tgz", strings.NewReader("DATA"), 4); err != nil {
		t.Fatal(err)
	}
	if method != "PUT" || path != "/api/packages/o/generic/tool/1.0/tool.tgz" || body != "DATA" || ctype != "application/octet-stream" {
		t.Errorf("upload: %s %s %q %s", method, path, body, ctype)
	}
	if got := c.PackageURL("tool", "1.0", "tool.tgz"); !strings.HasSuffix(got, "/api/packages/o/generic/tool/1.0/tool.tgz") {
		t.Errorf("url: %s", got)
	}
	if err := c.PackageLink(context.Background(), "tool"); err != nil || method != "POST" || path != "/api/v1/packages/o/generic/tool/-/link/r" {
		t.Errorf("link: %s %s %v", method, path, err)
	}
	if err := c.PackageDelete(context.Background(), "tool", "1.0"); err != nil || method != "DELETE" || path != "/api/v1/packages/o/generic/tool/1.0" {
		t.Errorf("delete: %s %s %v", method, path, err)
	}
	fail := newServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 500) })
	if err := fail.PackageUpload(context.Background(), "t", "1", "f", strings.NewReader("x"), 1); err == nil {
		t.Errorf("a 500 gave no error")
	}
}
