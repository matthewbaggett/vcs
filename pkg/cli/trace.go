package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const traceHelp = "Name the skill and step that sent the text, as skill or skill:step. $VCS_TRACE is the fallback. The command adds a hidden trace line to the end of the body."

var (
	claimLine   = regexp.MustCompile(`^⏳ claim host=[A-Za-z0-9._:?-]+ session=[A-Za-z0-9._:?-]+ pid=[A-Za-z0-9._:?-]+ since=[A-Za-z0-9._:?-]+$`)
	traceFirst  = regexp.MustCompile(`\A<!-- trace [^\n]* -->\n?`)
	traceLast   = regexp.MustCompile(`\n*<!-- trace [^\n]* -->\s*\z`)
	traceUnsafe = regexp.MustCompile(`[^A-Za-z0-9._:/?-]`)
	traceDashes = regexp.MustCompile(`-{2,}`)
)

// traceValue makes one safe token for a trace field. An empty value gives ?.
// A run of hyphens becomes one, so the value never holds the two hyphens that end an HTML comment.
func traceValue(s string) string {
	s = traceDashes.ReplaceAllString(traceUnsafe.ReplaceAllString(s, "_"), "-")
	if s == "" {
		return "?"
	}
	return s
}

// stampTrace puts one hidden line at the end of a body that a verb is about to send.
// It matches forgejo_trace_stamp of agent-standards. The skill and step come from
// --trace <skill>[:<step>], with $VCS_TRACE as the fallback. Call it after the prose lint,
// so the marker never reaches the linter. An empty body and an exact claim line stay as they are.
func stampTrace(env Env, flags map[string]string, body string) string {
	if body == "" || claimLine.MatchString(strings.TrimSuffix(body, "\n")) {
		return body
	}
	trace := flags["trace"]
	if trace == "" && env.Getenv != nil {
		trace = env.Getenv("VCS_TRACE")
	}
	skill, step, _ := strings.Cut(trace, ":")
	text := traceFirst.ReplaceAllString(body, "")
	text = traceLast.ReplaceAllString(text, "")
	text = strings.TrimRight(text, "\n")
	session := ""
	if env.Getenv != nil {
		session = env.Getenv("CLAUDE_CODE_SESSION_ID")
	}
	host := ""
	if env.Hostname != nil {
		host = env.Hostname()
	} else if h, err := os.Hostname(); err == nil {
		host, _, _ = strings.Cut(h, ".")
	}
	now := time.Now
	if env.Now != nil {
		now = env.Now
	}
	line := fmt.Sprintf("<!-- trace skill=%s step=%s session=%s host=%s at=%s -->\n",
		traceValue(skill), traceValue(step), traceValue(session), traceValue(host), now().UTC().Format("2006-01-02T15:04:05Z"))
	if text == "" {
		return line
	}
	return text + "\n\n" + line
}
