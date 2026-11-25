// Package session keeps a web session for a forge host. Some routes, such as rerun,
// take a session cookie and refuse a token.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CookieName is the name of the session cookie of Forgejo and Gitea.
const CookieName = "i_like_gitea"

// ErrNoSession means the host has no session, or the server no longer accepts it.
var ErrNoSession = errors.New("no session")

type file struct {
	Host  string `json:"host"`
	Value string `json:"i_like_gitea"`
}

// Path is where the session of a host lives, inside dir.
func Path(dir, host string) string {
	return filepath.Join(dir, "session."+host+".json")
}

// Save writes the cookie value of a host. The file is private to the user.
func Save(dir, host, value string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, _ := json.Marshal(file{Host: host, Value: value})
	return os.WriteFile(Path(dir, host), data, 0o600)
}

// Load reads the cookie value of a host. It returns ErrNoSession when there is none.
func Load(dir, host string) (string, error) {
	data, err := os.ReadFile(Path(dir, host))
	if err != nil {
		return "", ErrNoSession
	}
	var f file
	if json.Unmarshal(data, &f) != nil || f.Value == "" {
		return "", ErrNoSession
	}
	return f.Value, nil
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

// Check asks the server whether it accepts the cookie value.
func Check(ctx context.Context, baseURL, value string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/user/settings", nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: CookieName, Value: value})
	resp, err := client().Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: the server did not accept the cookie (HTTP %d)", ErrNoSession, resp.StatusCode)
	}
	return nil
}

// csrfField finds the hidden _csrf input of a login page. Older servers have one. Newer
// servers check the Origin and Sec-Fetch-Site headers instead and have none.
var csrfField = regexp.MustCompile(`<input[^>]*name="_csrf"[^>]*value="([^"]*)"`)

// sameOrigin marks a request as a form post from the page of the host itself.
func sameOrigin(req *http.Request, baseURL string) {
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/user/login")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// Login signs in with a user name and a password and returns the cookie value.
// An account with a second factor needs Import.
func Login(ctx context.Context, baseURL, user, password string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	c := client()
	c.Jar = jar
	// Get the login page first. It sets any CSRF cookie and holds the CSRF field.
	get, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/user/login", nil)
	if err != nil {
		return "", err
	}
	page, err := c.Do(get)
	if err != nil {
		return "", err
	}
	html, _ := io.ReadAll(io.LimitReader(page.Body, 1<<20))
	page.Body.Close()
	form := url.Values{"user_name": {user}, "password": {password}}
	if m := csrfField.FindSubmatch(html); m != nil {
		form.Set("_csrf", string(m[1]))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/user/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sameOrigin(req, baseURL)
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		return "", fmt.Errorf("the login was refused (HTTP %d). Check the user name and password", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if strings.Contains(loc, "/user/two_factor") || strings.Contains(loc, "/user/webauthn") {
		return "", fmt.Errorf("this account needs a second factor. Copy %s from a browser, then run: vcs session import", CookieName)
	}
	if strings.HasSuffix(strings.SplitN(loc, "?", 2)[0], "/user/login") {
		return "", errors.New("the login failed. Check the user name and password")
	}
	value := ""
	var names []string
	for _, ck := range resp.Cookies() {
		names = append(names, ck.Name)
		if ck.Name == CookieName {
			value = ck.Value
		}
	}
	if value == "" {
		return "", fmt.Errorf("the server set no %s cookie. It set: %s", CookieName, strings.Join(names, ", "))
	}
	if err := Check(ctx, baseURL, value); err != nil {
		return "", err
	}
	return value, nil
}
