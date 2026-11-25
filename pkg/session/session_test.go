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
	_, v, err := Login(context.Background(), srv.URL, "me", "pw")
	if err != nil || v != "abc123" {
		t.Fatalf("v=%q err=%v", v, err)
	}
	if _, _, err := Login(context.Background(), srv.URL, "me", "bad"); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("bad password: %v", err)
	}
	if err := Check(context.Background(), srv.URL, CookieName, "nope"); !errors.Is(err, ErrNoSession) {
		t.Errorf("bad cookie: %v", err)
	}
}

func TestLoginSecondFactor(t *testing.T) {
	srv := fakeForge(t, true)
	if _, _, err := Login(context.Background(), srv.URL, "me", "pw"); err == nil || !strings.Contains(err.Error(), "second factor") {
		t.Fatalf("err=%v", err)
	}
}

func TestSaveLoadRemove(t *testing.T) {
	dir := t.TempDir() + "/cfg"
	if _, _, err := Load(dir, "h"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("empty load: %v", err)
	}
	if err := Save(dir, "h", "i_like_forgejo", "val"); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(Path(dir, "h"))
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", st.Mode().Perm())
	}
	if n, v, err := Load(dir, "h"); err != nil || v != "val" || n != "i_like_forgejo" {
		t.Errorf("load %q %q %v", n, v, err)
	}
	if err := Remove(dir, "h"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "h"); err != nil {
		t.Errorf("second remove: %v", err)
	}
}

// loginForge serves a login that wants the same-origin headers. mode picks the answer:
// "ok" sets the session cookie under cookie, "refuse" redirects to the login page, and
// "nocookie" sets only other cookies.
func loginForge(t *testing.T, mode, cookie string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user/login":
			r.ParseForm()
			if r.Header.Get("Origin") == "" || r.Header.Get("Sec-Fetch-Site") != "same-origin" || mode == "refuse" {
				w.Header().Set("Location", "/user/login")
				w.WriteHeader(http.StatusSeeOther)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "lang", Value: "en"})
			if mode != "nocookie" {
				http.SetCookie(w, &http.Cookie{Name: cookie, Value: "abc123"})
			}
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		case r.URL.Path == "/user/settings":
			if c, err := r.Cookie(cookie); err == nil && c.Value == "abc123" {
				return
			}
			w.WriteHeader(http.StatusSeeOther)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginTakesTheCookieNameFromTheServer(t *testing.T) {
	for _, name := range []string{"session", "i_like_forgejo", "i_like_gitea", "my_own_session"} {
		srv := loginForge(t, "ok", name)
		n, v, err := Login(context.Background(), srv.URL, "me", "pw")
		if err != nil || n != name || v != "abc123" {
			t.Errorf("%s: n=%q v=%q err=%v", name, n, v, err)
		}
	}
}

func TestLoginRedirectToLoginIsAFailedLogin(t *testing.T) {
	srv := loginForge(t, "refuse", "i_like_forgejo")
	_, _, err := Login(context.Background(), srv.URL, "me", "pw")
	if err == nil || !strings.Contains(err.Error(), "login failed") || !strings.Contains(err.Error(), `Location "/user/login"`) {
		t.Fatalf("err=%v", err)
	}
}

func TestLoginReportNamesCookiesNotValues(t *testing.T) {
	srv := loginForge(t, "nocookie", "x")
	_, _, err := Login(context.Background(), srv.URL, "me", "pw")
	if err == nil || !strings.Contains(err.Error(), "cookies set: [lang]") || strings.Contains(err.Error(), "en]") {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckAny(t *testing.T) {
	srv := loginForge(t, "ok", "session")
	if n, err := CheckAny(context.Background(), srv.URL, "abc123"); err != nil || n != "session" {
		t.Errorf("n=%q err=%v", n, err)
	}
	if _, err := CheckAny(context.Background(), srv.URL, "bad"); !errors.Is(err, ErrNoSession) {
		t.Errorf("bad: %v", err)
	}
}

func TestLoadOldFileHasLegacyName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(Path(dir, "h"), []byte(`{"host":"h","i_like_gitea":"old"}`), 0o600)
	if n, v, err := Load(dir, "h"); err != nil || n != CookieName || v != "old" {
		t.Errorf("n=%q v=%q err=%v", n, v, err)
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
