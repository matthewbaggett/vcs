package cli

import (
	"fmt"
	"regexp"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/guard"
)

var labelColor = regexp.MustCompile(`^#?[0-9A-Fa-f]{6}$`)

// oneLine reports whether s is non-empty text of a single line.
func oneLine(s string) bool {
	return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\r\n")
}

// labelCreateCheck checks the arguments of label create. It returns an exit code, or -1 to go on.
func labelCreateCheck(env Env, v Verb, o opts) int {
	if len(o.pos) != 2 || !oneLine(o.pos[0]) || !labelColor.MatchString(o.pos[1]) || strings.ContainsAny(o.flags["desc"], "\r\n") {
		return usage(env, v)
	}
	if d := o.flags["desc"]; d != "" {
		if r := guard.Credit(d); len(r) > 0 {
			fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r[0])
			return ExitLint
		}
		if code := lintText(env, d); code != ExitOK {
			return code
		}
	}
	return -1
}
