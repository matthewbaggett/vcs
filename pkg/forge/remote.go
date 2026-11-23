package forge

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	urlRe = regexp.MustCompile(`^(?:https?|ssh)://(?:[^@/]+@)?([^/:]+)(?::[0-9]+)?/(.+)$`)
	scpRe = regexp.MustCompile(`^[^@/]+@([^:]+):(.+)$`)
)

// ParseRemote returns the host and the owner/name path of a git remote URL.
func ParseRemote(url string) (host, repo string, err error) {
	url = strings.TrimSpace(url)
	m := urlRe.FindStringSubmatch(url)
	if m == nil {
		m = scpRe.FindStringSubmatch(url)
	}
	if m == nil {
		return "", "", fmt.Errorf("cannot read the remote %q", url)
	}
	repo = strings.TrimSuffix(strings.Trim(m[2], "/"), ".git")
	if strings.Count(repo, "/") != 1 {
		return "", "", fmt.Errorf("the remote %q is not owner/name", url)
	}
	return m[1], repo, nil
}
