package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func fakeForge(t *testing.T, twoFactor bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/login":
			r.ParseForm()
			if r.PostForm.Get("user_name") != "me" || r.PostForm.Get("password") != "pw" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "abc123"})
			if twoFactor {
				w.Header().Set("Location", "/user/two_factor")
			} else {
				w.Header().Set("Location", "/")
			}
			w.WriteHeader(http.StatusSeeOther)
		case "/user/settings":
			if c, err := r.Cookie(CookieName); err == nil && c.Value == "abc123" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.Header().Set("Location", "/user/login")
			w.WriteHeader(http.StatusSeeOther)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginAndCheck(t *testing.T) {
	srv := fakeForge(t, false)
	v, err := Login(context.Background(), srv.URL, "me", "pw")
	if err != nil || v != "abc123" {
		t.Fatalf("v=%q err=%v", v, err)
	}
	if _, err := Login(context.Background(), srv.URL, "me", "bad"); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("bad password: %v", err)
	}
	if err := Check(context.Background(), srv.URL, "nope"); !errors.Is(err, ErrNoSession) {
		t.Errorf("bad cookie: %v", err)
	}
}

func TestLoginSecondFactor(t *testing.T) {
	srv := fakeForge(t, true)
	if _, err := Login(context.Background(), srv.URL, "me", "pw"); err == nil || !strings.Contains(err.Error(), "second factor") {
		t.Fatalf("err=%v", err)
	}
}

func TestSaveLoadRemove(t *testing.T) {
	dir := t.TempDir() + "/cfg"
	if _, err := Load(dir, "h"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("empty load: %v", err)
	}
	if err := Save(dir, "h", "val"); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(Path(dir, "h"))
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", st.Mode().Perm())
	}
	if v, err := Load(dir, "h"); err != nil || v != "val" {
		t.Errorf("load %q %v", v, err)
	}
	if err := Remove(dir, "h"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "h"); err != nil {
		t.Errorf("second remove: %v", err)
	}
}
