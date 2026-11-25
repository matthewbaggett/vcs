package session

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// KnownUser reads the user name of a host from an existing login of tea or fj. It returns ""
// when there is none. It reads the name only, never a token. home is the home directory.
func KnownUser(home, host string) string {
	if u := teaUser(filepath.Join(home, ".config", "tea", "config.yml"), host); u != "" {
		return u
	}
	return fjUser(filepath.Join(home, ".local", "share", "forgejo-cli", "keys.json"), host)
}

func teaUser(path, host string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		Logins []struct {
			URL  string `yaml:"url"`
			User string `yaml:"user"`
		} `yaml:"logins"`
	}
	if yaml.Unmarshal(data, &cfg) != nil {
		return ""
	}
	for _, l := range cfg.Logins {
		if u, err := url.Parse(l.URL); err == nil && strings.EqualFold(u.Host, host) && l.User != "" {
			return l.User
		}
	}
	return ""
}

func fjUser(path, host string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var keys struct {
		Hosts map[string]struct {
			Name string `json:"name"`
		} `json:"hosts"`
	}
	if json.Unmarshal(data, &keys) != nil {
		return ""
	}
	return keys.Hosts[host].Name
}
