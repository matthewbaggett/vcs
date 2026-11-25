package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// loginForge serves a login page that holds a CSRF field and a CSRF cookie. The post must
// give the field back, with the cookie and the same-origin headers.
func loginForge(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user/login" && r.Method == http.MethodGet:
			http.SetCookie(w, &http.Cookie{Name: "_csrf", Value: "tok"})
			w.Write([]byte(`<form><input type="hidden" name="_csrf" value="tok"></form>`))
		case r.URL.Path == "/user/login":
			r.ParseForm()
			c, err := r.Cookie("_csrf")
			ok := err == nil && c.Value == "tok" && r.PostForm.Get("_csrf") == "tok" &&
				r.Header.Get("Origin") != "" && r.Header.Get("Sec-Fetch-Site") == "same-origin"
			if !ok || mode == "refuse" {
				w.Header().Set("Location", "/user/login")
				w.WriteHeader(http.StatusSeeOther)
				return
			}
			if mode != "nocookie" {
				http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "abc123"})
			}
			http.SetCookie(w, &http.Cookie{Name: "lang", Value: "en"})
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		case r.URL.Path == "/user/settings":
			if c, err := r.Cookie(CookieName); err == nil && c.Value == "abc123" {
				return
			}
			w.WriteHeader(http.StatusSeeOther)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginSendsCSRFAndHeaders(t *testing.T) {
	srv := loginForge(t, "ok")
	v, err := Login(context.Background(), srv.URL, "me", "pw")
	if err != nil || v != "abc123" {
		t.Fatalf("v=%q err=%v", v, err)
	}
}

func TestLoginRedirectToLoginIsAFailedLogin(t *testing.T) {
	srv := loginForge(t, "refuse")
	_, err := Login(context.Background(), srv.URL, "me", "pw")
	if err == nil || !strings.Contains(err.Error(), "login failed") || strings.Contains(err.Error(), "no ") {
		t.Fatalf("err=%v", err)
	}
}

func TestLoginNamesTheCookiesItSaw(t *testing.T) {
	srv := loginForge(t, "nocookie")
	_, err := Login(context.Background(), srv.URL, "me", "pw")
	if err == nil || !strings.Contains(err.Error(), CookieName) || !strings.Contains(err.Error(), "lang") {
		t.Fatalf("err=%v", err)
	}
}

func TestKnownUser(t *testing.T) {
	home := t.TempDir()
	if got := KnownUser(home, "git.example"); got != "" {
		t.Errorf("none: %q", got)
	}
	fj := filepath.Join(home, ".local", "share", "forgejo-cli", "keys.json")
	os.MkdirAll(filepath.Dir(fj), 0o755)
	os.WriteFile(fj, []byte(`{"hosts":{"git.example":{"type":"Application","name":"fjuser","token":"secret"}}}`), 0o600)
	if got := KnownUser(home, "git.example"); got != "fjuser" {
		t.Errorf("fj: %q", got)
	}
	tea := filepath.Join(home, ".config", "tea", "config.yml")
	os.MkdirAll(filepath.Dir(tea), 0o755)
	os.WriteFile(tea, []byte("logins:\n  - name: a\n    url: https://other.example\n    user: nope\n  - name: b\n    url: https://git.example\n    user: teauser\n"), 0o600)
	if got := KnownUser(home, "git.example"); got != "teauser" {
		t.Errorf("tea: %q", got)
	}
	if got := KnownUser(home, "missing.example"); got != "" {
		t.Errorf("other host: %q", got)
	}
}
