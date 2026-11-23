// Package guard holds the version control rules of hooks/vcs-guard.sh as Go checks. Each check
// returns the reasons it refuses, or nothing.
package guard

import (
	"regexp"
	"strings"
)

var (
	typeRe    = regexp.MustCompile(`^(feature|fix|docs|chore|refactor|test|ci|perf|build|revert)/[a-z0-9]+(-[a-z0-9]+)*$`)
	creditRe  = regexp.MustCompile(`(?im)^\s*(co-authored-by:.*(claude|anthropic|copilot|gpt|openai|gemini|\bai\b|noreply@anthropic\.com)|.*generated with \[?claude|.*🤖)`)
	protected = map[string]bool{"main": true, "master": true}
)

// Branch checks a branch name against the type/short-description rule.
func Branch(name string) []string {
	if !typeRe.MatchString(name) {
		return []string{"branch " + quote(name) + " does not match type/short-description. The types are feature, fix, docs, chore, refactor, test, ci, perf, build, revert"}
	}
	return nil
}

// Commit checks a commit on a branch, with its message.
func Commit(branch, message string) []string {
	var out []string
	if protected[branch] {
		out = append(out, "never commit to "+branch+". Work on a branch")
	}
	out = append(out, Credit(message)...)
	return out
}

// Credit refuses an AI credit line in a commit message or a pull request body.
func Credit(text string) []string {
	if creditRe.MatchString(text) {
		return []string{"the text has an AI credit line. Add none to a commit message or a pull request body"}
	}
	return nil
}

// Push checks the arguments of a git push.
func Push(args []string) []string {
	var out []string
	for _, a := range args {
		switch {
		case a == "-f" || a == "--force" || strings.HasPrefix(a, "--force-with-lease") || (strings.HasPrefix(a, "+") && len(a) > 1):
			out = append(out, "never force-push")
		case a == "--no-verify":
			out = append(out, "never skip the hooks with --no-verify")
		}
	}
	return out
}

func quote(s string) string { return `"` + s + `"` }
