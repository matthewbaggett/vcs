package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// secretName is the form that Forgejo accepts for a secret name.
var secretName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// secretSetCheck reads the value of "secret set" from a file, from standard input, or from an
// environment variable. The value never comes from an argument, so it stays out of the process
// list and the shell history. The text goes in o.flags["value"]. It returns -1 to go on.
func secretSetCheck(env Env, v Verb, o *opts) int {
	vf, fe := o.flags["value-file"], o.flags["from-env"]
	if len(o.pos) != 1 || !secretName.MatchString(o.pos[0]) || (vf == "") == (fe == "") {
		return usage(env, v)
	}
	var val string
	if fe != "" {
		val = env.Getenv(fe)
	} else {
		var raw []byte
		var err error
		if vf == "-" {
			if env.Stdin == nil {
				return usage(env, v)
			}
			raw, err = io.ReadAll(env.Stdin)
		} else {
			raw, err = os.ReadFile(vf)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: cannot read a secret value from %s\n", vf)
			return ExitUsage
		}
		val = string(raw)
	}
	val = strings.TrimRight(val, "\r\n")
	if val == "" {
		fmt.Fprintln(env.Stderr, "vcs: the secret value is empty. Nothing was sent")
		return ExitUsage
	}
	o.flags["value"] = val
	return -1
}

// branchName is the form of a branch name or a rule pattern that "protection list" accepts.
var branchName = regexp.MustCompile(`^[0-9A-Za-z._/@+-]+$`)

// protectionCovers says whether a rule covers a branch. A rule covers it when the rule name equals
// the branch, or the rule name is a pattern that matches it. A star stops at a slash. Two stars do not.
func protectionCovers(rule, branch string) bool {
	if rule == "" {
		return false
	}
	if rule == branch {
		return true
	}
	var b strings.Builder
	b.WriteString("^")
	for i, part := range strings.Split(rule, "**") {
		if i > 0 {
			b.WriteString(".*")
		}
		for j, seg := range strings.Split(part, "*") {
			if j > 0 {
				b.WriteString("[^/]*")
			}
			b.WriteString(regexp.QuoteMeta(seg))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(branch)
}
