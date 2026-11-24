package forgejo

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func TestProtectionList(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/branch_protections" {
			t.Errorf("unexpected %s", r.URL)
		}
		w.Write([]byte(`[
		 {"rule_name":"main","enable_status_check":true,"status_check_contexts":["CI / a (pull_request)","b"],"required_approvals":1,"enable_push":false,"apply_to_admins":true},
		 {"branch_name":"old","enable_status_check":false,"status_check_contexts":["x"],"enable_push":true,"enable_push_whitelist":true},
		 {"rule_name":"rel/*","enable_push":true}]`))
	})
	ps, err := c.ProtectionList(context.Background())
	if err != nil || len(ps) != 3 {
		t.Fatalf("rules: %+v %v", ps, err)
	}
	if ps[0].Rule != "main" || len(ps[0].Contexts) != 2 || ps[0].Approvals != 1 || ps[0].Push != "none" || !ps[0].Admins {
		t.Errorf("first: %+v", ps[0])
	}
	if ps[1].Rule != "old" || len(ps[1].Contexts) != 0 || ps[1].Push != "whitelist" {
		t.Errorf("second: %+v", ps[1])
	}
	if ps[2].Push != "all" {
		t.Errorf("third: %+v", ps[2])
	}
}

func TestProtectionListRefused(t *testing.T) {
	for _, code := range []int{401, 403} {
		c := newServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", code) })
		if _, err := c.ProtectionList(context.Background()); !errors.Is(err, forge.ErrNoSession) {
			t.Errorf("%d: %v", code, err)
		}
	}
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	if _, err := c.ProtectionList(context.Background()); err == nil {
		t.Errorf("a non-list body gave no error")
	}
}
