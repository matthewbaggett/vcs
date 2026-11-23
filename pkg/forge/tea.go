package forge

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// TeaToken returns the API token that the tea login for a host holds. It reads the config file of
// tea and nothing else. It never writes. It returns an empty string when the file, the host, or
// the token is missing. When two logins match the host, the one that tea marks as default wins.
func TeaToken(host string) string {
	for _, path := range teaConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if tok := parseTeaToken(string(data), host); tok != "" {
			return tok
		}
	}
	return ""
}

func teaConfigPaths() []string {
	if p := os.Getenv("TEA_CONFIG"); p != "" {
		return []string{p}
	}
	var out []string
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		out = append(out, filepath.Join(x, "tea", "config.yml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".config", "tea", "config.yml"))
	}
	if dir, err := os.UserConfigDir(); err == nil {
		out = append(out, filepath.Join(dir, "tea", "config.yml"))
	}
	return out
}

// parseTeaToken reads the logins list of a tea config. Each login starts with a `- name:` line.
func parseTeaToken(data, host string) string {
	type login struct{ url, token, def string }
	var logins []login
	cur := -1
	for _, line := range strings.Split(data, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") {
			t = strings.TrimSpace(strings.TrimPrefix(t, "- "))
			if strings.HasPrefix(t, "name:") {
				logins = append(logins, login{})
				cur = len(logins) - 1
			}
		}
		if cur < 0 {
			continue
		}
		k, v, ok := strings.Cut(t, ":")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimSpace(k) {
		case "url":
			logins[cur].url = v
		case "token":
			logins[cur].token = v
		case "default":
			logins[cur].def = v
		}
	}
	best := ""
	for _, l := range logins {
		u, err := url.Parse(l.url)
		if err != nil || l.token == "" || !strings.EqualFold(u.Host, host) {
			continue
		}
		if l.def == "true" {
			return l.token
		}
		if best == "" {
			best = l.token
		}
	}
	return best
}

// ForgejoToken picks the token for a Forgejo or Gitea host. VCS_TOKEN wins. Without it, the tea
// login for the host gives the token. It returns an empty string when neither holds one.
func ForgejoToken(getenv func(string) string, host string) string {
	if t := getenv("VCS_TOKEN"); t != "" {
		return t
	}
	return TeaToken(host)
}
