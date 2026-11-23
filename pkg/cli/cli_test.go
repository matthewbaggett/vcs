package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type fake struct {
	pr   forge.PRStatus
	runs []forge.Run
	jobs []forge.Job
	log  string
	err  error
}

func (f fake) PRStatus(context.Context, int) (forge.PRStatus, error) { return f.pr, f.err }
func (f fake) CIRuns(context.Context, forge.RunFilter) ([]forge.Run, error) {
	return f.runs, f.err
}
func (f fake) CIJobs(context.Context, int64) ([]forge.Job, error) { return f.jobs, f.err }
func (f fake) CILog(context.Context, int64) (string, error)       { return f.log, f.err }

func run(t *testing.T, c forge.Client, env map[string]string, tty bool, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	e := Env{
		Stdout: &out, Stderr: &errb, IsTerminal: tty,
		Getenv:    func(k string) string { return env[k] },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(host, repo string) (forge.Client, error) {
			if host == "github.com" {
				return nil, forge.ErrUnsupported
			}
			return c, nil
		},
	}
	code := Run(context.Background(), args, e)
	return code, out.String(), errb.String()
}

func TestPRStatusRow(t *testing.T) {
	c := fake{pr: forge.PRStatus{State: "open", Draft: true, CI: "failure", HeadSHA: "abc", URL: "u", ConflictedFiles: []string{"a", "b"}}}
	code, out, errs := run(t, c, nil, true, "pr", "status", "12")
	if code != 0 || out != "open\ttrue\tfalse\ta,b\tfailure\tabc\tu\n" || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestAgentHintOnlyForAgent(t *testing.T) {
	c := fake{pr: forge.PRStatus{State: "open"}}
	_, _, human := run(t, c, nil, true, "pr", "status", "12")
	if strings.Contains(human, "vcs: next:") {
		t.Errorf("a person saw a hint: %q", human)
	}
	for name, env := range map[string]map[string]string{
		"flag-env":   {"VCS_CALLER": "agent"},
		"claudecode": {"CLAUDECODE": "1"},
	} {
		_, _, errs := run(t, c, env, true, "pr", "status", "12")
		if !strings.Contains(errs, "vcs: next: ") {
			t.Errorf("%s: no hint: %q", name, errs)
		}
	}
	// No terminal means an agent. VCS_CALLER=human forces the other way.
	if _, _, errs := run(t, c, nil, false, "pr", "status", "12"); !strings.Contains(errs, "vcs: next:") {
		t.Errorf("pipe: no hint: %q", errs)
	}
	if _, _, errs := run(t, c, map[string]string{"VCS_CALLER": "human"}, false, "pr", "status", "12"); errs != "" {
		t.Errorf("human forced: %q", errs)
	}
}

func TestUsageErrorsExit64(t *testing.T) {
	for _, args := range [][]string{
		{}, {"pr"}, {"pr", "status"}, {"pr", "status", "x"}, {"pr", "nope"},
		{"ci", "runs", "--limit", "x"}, {"ci", "runs", "--bogus", "1"}, {"ci", "runs", "--sha", "zz"},
		{"ci", "jobs"}, {"ci", "jobs", "abc"}, {"ci", "log", "0"}, {"ci", "log", "5", "--tail", "-1"},
		{"docs", "gen"},
	} {
		if code, _, _ := run(t, fake{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v exit %d, want 64", args, code)
		}
	}
}

func TestExitCodes(t *testing.T) {
	if code, _, errs := run(t, fake{}, nil, true, "--host", "github.com", "pr", "status", "1"); code != ExitUnavailable || !strings.Contains(errs, "github.com") {
		t.Errorf("github: %d %q", code, errs)
	}
	if code, _, _ := run(t, fake{err: errors.New("boom")}, nil, true, "pr", "status", "1"); code != ExitFail {
		t.Errorf("fail: %d", code)
	}
}

func TestCIRunsAndJobs(t *testing.T) {
	c := fake{
		runs: []forge.Run{{ID: 9, Index: 4, Status: "failure", Ref: "#3", SHA: "s", Title: "t", URL: "https://git.example/o/r/actions/runs/4"}},
		jobs: []forge.Job{{ID: 7, Name: "w", Status: "success", Attempt: 1, NoLog: true}},
	}
	if _, out, _ := run(t, c, nil, true, "ci", "runs"); out != "9\t4\tfailure\t#3\ts\tt\thttps://git.example/o/r/actions/runs/4\n" {
		t.Errorf("runs: %q", out)
	}
	if _, out, _ := run(t, c, nil, true, "ci", "jobs", "9"); out != "7\tw\tsuccess\t1\tno-log\n" {
		t.Errorf("jobs: %q", out)
	}
	// A run address holds the index. The command finds the id.
	if code, out, _ := run(t, c, nil, true, "ci", "jobs", "https://git.example/o/r/actions/runs/4"); code != 0 || out == "" {
		t.Errorf("jobs by address: %d %q", code, out)
	}
	if code, _, errs := run(t, c, nil, true, "ci", "jobs", "https://git.example/o/r/actions/runs/99"); code != ExitFail || !strings.Contains(errs, "99") {
		t.Errorf("unknown address: %d %q", code, errs)
	}
}

func TestCILogStripsColourAndTails(t *testing.T) {
	c := fake{log: "one\n\x1b[31mtwo\x1b[0m\nthree\n"}
	if _, out, _ := run(t, c, nil, true, "ci", "log", "5", "--tail", "2"); out != "two\nthree\n" {
		t.Errorf("tail: %q", out)
	}
	if _, out, _ := run(t, c, nil, true, "ci", "log", "5"); out != "one\ntwo\nthree\n" {
		t.Errorf("all: %q", out)
	}
	code, _, errs := run(t, fake{err: forge.ErrNoLog}, nil, true, "ci", "log", "5")
	if code != ExitFail || !strings.Contains(errs, "wrapper") {
		t.Errorf("no log: %d %q", code, errs)
	}
}

// Every verb needs an agent hint for each outcome. An empty text is allowed, a missing key is not.
func TestEveryVerbHasEveryHint(t *testing.T) {
	for _, v := range Tree {
		for _, out := range []string{OutOK, OutFail, OutUsage, OutUnavailable} {
			if _, ok := v.Hints[out]; !ok {
				t.Errorf("%s %s has no hint for %q", v.Noun, v.Name, out)
			}
		}
		if v.Usage == "" || v.Summary == "" || v.Output == "" || len(v.Exits) == 0 {
			t.Errorf("%s %s is not documented", v.Noun, v.Name)
		}
	}
}

// The committed pages must match the command tree. Run `go run ./cmd/vcs docs gen docs/Commands` to fix a drift.
func TestCommittedDocsMatchTree(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "Commands")
	for name, want := range Pages() {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Errorf("docs/Commands/%s is out of date. Run: go run ./cmd/vcs docs gen docs/Commands", name)
		}
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if _, ok := Pages()[e.Name()]; !ok && e.Name() != "README.md" {
			t.Errorf("docs/Commands/%s has no verb", e.Name())
		}
	}
}
