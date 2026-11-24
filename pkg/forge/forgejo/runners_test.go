package forgejo

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func TestRunnerList(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/admin/actions/runners" {
			t.Errorf("unexpected %s", r.URL)
		}
		w.Write([]byte(`[{"id":9,"name":"b","status":"idle","uuid":"u9","version":"v6","labels":[{"name":"x"},{"name":"y"}]},{"id":2,"name":"a","status":"offline","uuid":"u2","version":"v5","labels":[]}]`))
	})
	rs, err := c.RunnerList(context.Background())
	if err != nil || len(rs) != 2 || rs[0].ID != 2 || rs[1].Labels[1] != "y" || rs[1].UUID != "u9" {
		t.Fatalf("runners: %+v %v", rs, err)
	}
}

func TestRunnerListRefused(t *testing.T) {
	for _, code := range []int{401, 403} {
		c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "token does not have the admin scope", code)
		})
		if _, err := c.RunnerList(context.Background()); !errors.Is(err, forge.ErrNoSession) {
			t.Errorf("%d: %v", code, err)
		}
	}
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	if _, err := c.RunnerList(context.Background()); err == nil {
		t.Errorf("a non-list body gave no error")
	}
}
