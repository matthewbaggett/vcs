package forge

import (
	"os"
	"path/filepath"
	"strings"
)

// NetrcLogin returns the login and password that ~/.netrc holds for a host.
// It returns empty strings when the file or the host is missing.
func NetrcLogin(host string) (login, password string) {
	path := os.Getenv("NETRC")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", ""
		}
		path = filepath.Join(home, ".netrc")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	return parseNetrc(string(data), host)
}

func parseNetrc(data, host string) (login, password string) {
	toks := strings.Fields(data)
	in := false
	for i := 0; i < len(toks); i++ {
		switch toks[i] {
		case "machine":
			if i+1 < len(toks) {
				i++
				in = toks[i] == host
			}
		case "default":
			in = true
		case "login":
			if i+1 < len(toks) {
				i++
				if in && login == "" {
					login = toks[i]
				}
			}
		case "password":
			if i+1 < len(toks) {
				i++
				if in && password == "" {
					password = toks[i]
				}
			}
		}
	}
	return login, password
}
