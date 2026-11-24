package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/lint"
)

// keepTrace lets one test read the trace line. Other tests see the body without it.
var keepTrace bool

func unstamp(s string) string {
	if keepTrace {
		return s
	}
	return strings.TrimRight(traceLast.ReplaceAllString(s, ""), "\n")
}

// seeChecker records each text that the prose lint reads.
type seeChecker struct{ seen *[]string }

func (s seeChecker) Name() string { return "see" }
func (s seeChecker) Check(text string) ([]lint.Finding, error) {
	*s.seen = append(*s.seen, text)
	return nil, nil
}

func traceRun(t *testing.T, sink *string, seen *[]string, env map[string]string, args ...string) int {
	t.Helper()
	keepTrace = true
	t.Cleanup(func() { keepTrace = false })
	var out, errb bytes.Buffer
	e := Env{
		Stdout: &out, Stderr: &errb,
		Getenv:    func(k string) string { return env[k] },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return commenterFor(sink), nil },
		Now:       func() time.Time { return time.Date(2026, 11, 24, 1, 2, 3, 0, time.UTC) },
		Hostname:  func() string { return "box" },
	}
	if seen != nil {
		e.Checkers = []lint.Checker{seeChecker{seen: seen}}
	}
	return Run(context.Background(), args, e)
}

func commenterFor(sink *string) forge.Client {
	return commenter{last: sink, fake: fake{comment: sink}}
}

const stamped = "<!-- trace skill=pr-patrol step=3 session=s1 host=box at=2026-11-24T01:02:03Z -->"

func TestTraceStampEachVerb(t *testing.T) {
	env := map[string]string{"CLAUDE_CODE_SESSION_ID": "s1"}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"pr", "create", "T", "--head", "fix/a", "--body", "Text."}, "fix/a<-main|T|Text.\n\n" + stamped + "\n"},
		{[]string{"pr", "edit", "5", "--body", "Text."}, "edit 5|-|Text.\n\n" + stamped + "\n"},
		{[]string{"issue", "create", "T", "--body", "Text."}, "T|Text.\n\n" + stamped + "\n"},
		{[]string{"issue", "edit", "5", "--body", "Text."}, "issue 5|-|Text.\n\n" + stamped + "\n|-|-|[]|[]"},
		{[]string{"comment", "add", "5", "--body", "Text."}, "Text.\n\n" + stamped + "\n"},
		{[]string{"comment", "edit", "5", "--body", "Text."}, "edit 5|Text.\n\n" + stamped + "\n"},
	} {
		var sink string
		args := append(tc.args, "--trace", "pr-patrol:3")
		if code := traceRun(t, &sink, nil, env, args...); code != ExitOK || sink != tc.want {
			t.Errorf("%v: code=%d sink=%q want=%q", tc.args[:2], code, sink, tc.want)
		}
	}
}

func TestTraceStampRules(t *testing.T) {
	env := map[string]string{"VCS_TRACE": "env-skill"}
	var sink string
	// The environment is the fallback, and an empty field is ?.
	traceRun(t, &sink, nil, env, "comment", "add", "5", "--body", "Text.")
	if want := "Text.\n\n<!-- trace skill=env-skill step=? session=? host=box at=2026-11-24T01:02:03Z -->\n"; sink != want {
		t.Errorf("fallback: %q", sink)
	}
	// An old trace line goes, first or last, and the body never holds two.
	old := "<!-- trace skill=a step=b session=c host=d at=e -->"
	for _, body := range []string{old + "\nText.", "Text.\n\n" + old + "\n"} {
		traceRun(t, &sink, nil, nil, "comment", "add", "5", "--body", body)
		if strings.Count(sink, "<!-- trace") != 1 || !strings.HasPrefix(sink, "Text.\n\n<!-- trace skill=?") {
			t.Errorf("old line: %q", sink)
		}
	}
	// An exact claim line gets no trace line.
	claim := "⏳ claim host=box session=s1 pid=42 since=2026-11-24T01:00:00Z"
	traceRun(t, &sink, nil, nil, "comment", "add", "5", "--body", claim)
	if sink != claim {
		t.Errorf("claim: %q", sink)
	}
	// A hyphen run and a stray character stay safe in the comment.
	traceRun(t, &sink, nil, nil, "comment", "add", "5", "--body", "Text.", "--trace", "a--b c:d--e")
	if !strings.Contains(sink, "skill=a-b_c step=d-e ") {
		t.Errorf("safe token: %q", sink)
	}
	// A title-only edit sends no body, so it has no trace line.
	traceRun(t, &sink, nil, nil, "pr", "edit", "5", "--title", "New")
	if sink != "edit 5|New|-" {
		t.Errorf("title only: %q", sink)
	}
}

func TestTraceStampComesAfterTheLint(t *testing.T) {
	var sink string
	var seen []string
	traceRun(t, &sink, &seen, nil, "comment", "add", "5", "--body", "Text.", "--trace", "x")
	if len(seen) != 1 || seen[0] != "Text." || !strings.Contains(sink, "<!-- trace skill=x") {
		t.Errorf("lint saw %q, sent %q", seen, sink)
	}
}
