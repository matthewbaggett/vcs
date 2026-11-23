package cli

import (
	"fmt"
	"regexp"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/guard"
)

var milestoneDue = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// milestoneCheck checks the arguments of the milestone verbs. It returns an exit code, or -1 to go on.
func milestoneCheck(env Env, v Verb, o opts) int {
	switch v.Name {
	case "list":
		if s := o.flags["state"]; len(o.pos) != 0 || (s != "" && s != "open" && s != "closed" && s != "all") {
			return usage(env, v)
		}
	case "view":
		if len(o.pos) != 1 || !singleLine(o.pos[0]) {
			return usage(env, v)
		}
	case "create":
		d := o.flags["desc"]
		if len(o.pos) != 1 || !singleLine(o.pos[0]) || (o.flags["due"] != "" && !milestoneDue.MatchString(o.flags["due"])) {
			return usage(env, v)
		}
		if r := guard.Credit(o.pos[0] + "\n" + d); len(r) > 0 {
			fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r[0])
			return ExitLint
		}
		if code := lintText(env, o.pos[0]); code != ExitOK {
			return code
		}
		if strings.TrimSpace(d) != "" {
			if code := lintText(env, d); code != ExitOK {
				return code
			}
		}
	}
	return -1
}

// singleLine reports whether s is non-empty text of a single line.
func singleLine(s string) bool {
	return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\r\n")
}
