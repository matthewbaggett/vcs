// Package session keeps a web session for a forge host. Some routes, such as rerun,
// take a session cookie and refuse a token.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CookieName is the legacy name of the session cookie. Gitea uses it, and so did older
// Forgejo. A session saved with no name has this one.
const CookieName = "i_like_gitea"

// Candidates are the names of the session cookie that vcs tries first. git.grey.ooo names it session. Forgejo has used
// i_like_forgejo since release 1.20. The server setting [session] COOKIE_NAME can change it.
var Candidates = []string{"session", "i_like_forgejo", CookieName}

// ErrNoSession means the host has no session, or the server no longer accepts it.
var ErrNoSession = errors.New("no session")

type file struct {
	Host string `json:"host"`
	// Name is the cookie name. Older files have none, and then it is CookieName.
	Name  string `json:"name,omitempty"`
	Value string `json:"i_like_gitea"`
}

// Path is where the session of a host lives, inside dir.
func Path(dir, host string) string {
	return filepath.Join(dir, "session."+host+".json")
}

// Save writes the cookie name and value of a host. The file is private to the user.
func Save(dir, host, name, value string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, _ := json.Marshal(file{Host: host, Name: name, Value: value})
	return os.WriteFile(Path(dir, host), data, 0o600)
}

// Load reads the cookie name and value of a host. It returns ErrNoSession when there is none.
func Load(dir, host string) (name, value string, err error) {
	data, err := os.ReadFile(Path(dir, host))
	if err != nil {
		return "", "", ErrNoSession
	}
	var f file
	if json.Unmarshal(data, &f) != nil || f.Value == "" {
		return "", "", ErrNoSession
	}
	if f.Name == "" {
		f.Name = CookieName
	}
	return f.Name, f.Value, nil
}

// Remove deletes the session of a host. A missing session is not an error.
func Remove(dir, host string) error {
	err := os.Remove(Path(dir, host))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func client() *http.Client {
	return &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// Check asks the server whether it accepts the cookie.
func Check(ctx context.Context, baseURL, name, value string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/user/settings", nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: name, Value: value})
	resp, err := client().Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: the server did not accept the cookie %s (HTTP %d)", ErrNoSession, name, resp.StatusCode)
	}
	return nil
}

// CheckAny tries a bare cookie value under each candidate name and returns the name that the
// server accepts.
func CheckAny(ctx context.Context, baseURL, value string) (string, error) {
	var last error
	for _, n := range Candidates {
		if last = Check(ctx, baseURL, n, value); last == nil {
			return n, nil
		}
	}
	return "", fmt.Errorf("%w: the server accepted the value under none of %s", ErrNoSession, strings.Join(Candidates, ", "))
}

// sameOrigin marks a request as a form post from the page of the host itself.
func sameOrigin(req *http.Request, baseURL string) {
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/user/login")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// notSession names cookies that never hold the login.
var notSession = map[string]bool{"persistent": true, "redirect_to": true, "lang": true, "_csrf": true, "_flash": true, "gitea_incredible": true, "macaron_flash": true}

// Login signs in with a user name and a password and returns the name and value of the
// session cookie. The name comes from the server. An account with a second factor needs Import.
func Login(ctx context.Context, baseURL, user, password string) (string, string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	form := url.Values{"user_name": {user}, "password": {password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/user/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sameOrigin(req, baseURL)
	resp, err := client().Do(req)
	if err != nil {
		return "", "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	var set []*http.Cookie
	var names []string
	for _, ck := range resp.Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			continue
		}
		names = append(names, ck.Name)
		if !notSession[ck.Name] {
			set = append(set, ck)
		}
	}
	// The report names the redirect and the cookie names. It never holds a value.
	report := fmt.Sprintf("HTTP %d, Location %q, cookies set: [%s]", resp.StatusCode, loc, strings.Join(names, " "))
	if resp.StatusCode != http.StatusSeeOther {
		return "", "", fmt.Errorf("the login was refused (%s). Check the user name and password", report)
	}
	if strings.Contains(loc, "/user/two_factor") || strings.Contains(loc, "/user/webauthn") {
		return "", "", errors.New("this account needs a second factor. Copy the session cookie from a browser, then run: vcs session import")
	}
	if strings.HasSuffix(strings.SplitN(loc, "?", 2)[0], "/user/login") {
		return "", "", fmt.Errorf("the login failed (%s). Check the user name and password", report)
	}
	// Try the known names first, then every other cookie. The server must accept the pair.
	for _, want := range append(append([]string{}, Candidates...), "") {
		for _, ck := range set {
			if (want == "" && contains(Candidates, ck.Name)) || (want != "" && ck.Name != want) {
				continue
			}
			if err := Check(ctx, baseURL, ck.Name, ck.Value); err == nil {
				return ck.Name, ck.Value, nil
			}
		}
	}
	return "", "", fmt.Errorf("the server set no session cookie that it accepts (%s)", report)
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
