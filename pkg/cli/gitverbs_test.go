package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gitSim stands in for git. It records each call that changes something, and it answers the
// queries from fields.
type gitSim struct {
	branch   string
	upstream string
	top      string
	held     bool // origin holds the branch
	hasMain  bool
	calls    []string
	fail     map[string]int   // a call prefix to an exit code
	dir      string           // the last -C directory
	dirty    bool             // the tree has changes
	merging  bool             // a merge is in progress
	scripts  map[string][]int // a project script to its successive exit codes. The last one repeats
	ran      []string         // each project script that ran, with its arguments
}

func (s *gitSim) exec(_ context.Context, name string, args []string, stdout, _ io.Writer) int {
	if strings.HasPrefix(name, s.top) && s.scripts != nil {
		base := filepath.Base(name)
		s.ran = append(s.ran, base+" "+strings.Join(args, " "))
		codes := s.scripts[base]
		if len(codes) == 0 {
			return 0
		}
		c := codes[0]
		if len(codes) > 1 {
			s.scripts[base] = codes[1:]
		}
		return c
	}
	if name == "git" && len(args) >= 2 && args[0] == "-C" {
		s.dir, args = args[1], args[2:]
	}
	line := name + " " + strings.Join(args, " ")
	switch {
	case line == "git status --porcelain --untracked-files=no":
		if s.dirty {
			io.WriteString(stdout, " M file\n")
		}
		return 0
	case line == "git rev-parse --verify --quiet MERGE_HEAD":
		if s.merging {
			return 0
		}
		return 1
	case line == "git rev-parse --verify --quiet HEAD":
		io.WriteString(stdout, "abc123\n")
		return 0
	case line == "git symbolic-ref --quiet --short HEAD":
		if s.branch == "" {
			return 1
		}
		io.WriteString(stdout, s.branch+"\n")
		return 0
	case line == "git rev-parse --show-toplevel":
		io.WriteString(stdout, s.top+"\n")
		return 0
	case strings.HasPrefix(line, "git rev-parse --abbrev-ref"):
		if s.upstream == "" {
			return 128
		}
		io.WriteString(stdout, s.upstream+"\n")
		return 0
	case strings.HasPrefix(line, "git ls-remote"):
		if s.held {
			return 0
		}
		return 2
	case strings.HasPrefix(line, "git rev-parse --verify"):
		if s.hasMain {
			return 0
		}
		return 1
	}
	s.calls = append(s.calls, line)
	for p, c := range s.fail {
		if strings.HasPrefix(line, p) {
			return c
		}
	}
	return 0
}

func gitRun(t *testing.T, s *gitSim, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errb, Getenv: func(string) string { return "" }, Exec: s.exec}
	code := Run(context.Background(), append([]string{"git"}, args...), env)
	return code, out.String(), errb.String()
}

func TestGitCommit(t *testing.T) {
	s := &gitSim{branch: "feature/x", top: t.TempDir()}
	code, out, _ := gitRun(t, s, "commit", "-m", "Add the thing")
	if code != ExitOK || len(s.calls) != 1 || s.calls[0] != "git commit -m Add the thing" || !strings.Contains(out, "check not-default-branch: ok") {
		t.Fatalf("good: %d %v %q", code, s.calls, out)
	}
	for _, c := range []struct {
		name   string
		branch string
		args   []string
		code   int
	}{
		{"no-verify", "feature/x", []string{"commit", "-m", "A", "--no-verify"}, ExitUsage},
		{"short n", "feature/x", []string{"commit", "-nm", "A"}, ExitUsage},
		{"main", "main", []string{"commit", "-m", "A"}, ExitFail},
		{"bad name", "wip", []string{"commit", "-m", "A"}, ExitFail},
		{"credit", "feature/x", []string{"commit", "-m", "Add it", "-m", "Co-Authored-By: Claude <noreply@anthropic.com>"}, ExitLint},
		{"credit joined", "feature/x", []string{"commit", "-mDone\n\n🤖 Generated with Claude"}, ExitLint},
	} {
		s := &gitSim{branch: c.branch, top: t.TempDir()}
		if code, _, _ := gitRun(t, s, c.args...); code != c.code || len(s.calls) != 0 {
			t.Errorf("%s: code %d calls %v", c.name, code, s.calls)
		}
	}
	// -m with a later n is a message, not a hook bypass.
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "commit", "-m", "run", "--amend"); code != ExitOK {
		t.Errorf("amend: %d", code)
	}
	// A message from a file, and an amend with no message at all.
	f := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(f, []byte("Fine message\n\n🤖 Generated with Claude\n"), 0o644)
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "commit", "-F", f); code != ExitLint {
		t.Errorf("file credit: %d", code)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, out, _ := gitRun(t, s, "commit", "--amend", "--no-edit"); code != ExitOK || !strings.Contains(out, "attribution: skipped") {
		t.Errorf("no message: %d %q", code, out)
	}
	// git failing gives its own exit code.
	s = &gitSim{branch: "feature/x", top: t.TempDir(), fail: map[string]int{"git commit": 1}}
	if code, _, _ := gitRun(t, s, "commit", "-m", "A"); code != 1 {
		t.Errorf("git fail: %d", code)
	}
}

func TestGitProjectChecks(t *testing.T) {
	top := t.TempDir()
	os.MkdirAll(filepath.Join(top, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(top, "scripts"), 0o755)
	os.WriteFile(filepath.Join(top, ".claude", "check-wiring.sh"), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(top, "scripts", "vcs-checks.sh"), []byte("#!/bin/sh\n"), 0o755)
	var ran []string
	var failOn string
	var out, errb bytes.Buffer
	sim := &gitSim{branch: "feature/x", top: top}
	env := Env{Stdout: &out, Stderr: &errb, Getenv: func(string) string { return "" },
		Exec: func(ctx context.Context, name string, args []string, so, se io.Writer) int {
			if strings.HasPrefix(name, top) {
				ran = append(ran, filepath.Base(name)+" "+strings.Join(args, " "))
				if filepath.Base(name) == failOn {
					return 1
				}
				return 0
			}
			return sim.exec(ctx, name, args, so, se)
		}}
	if code := Run(context.Background(), []string{"git", "commit", "-m", "A"}, env); code != ExitOK || strings.Join(ran, "|") != "check-wiring.sh --ci|vcs-checks.sh commit" {
		t.Fatalf("both: %d %v", code, ran)
	}
	failOn = "check-wiring.sh"
	sim.calls = nil
	if code := Run(context.Background(), []string{"git", "commit", "-m", "A"}, env); code != ExitFail || len(sim.calls) != 0 {
		t.Errorf("wiring fail: %d %v", code, sim.calls)
	}
	failOn = "vcs-checks.sh"
	if code := Run(context.Background(), []string{"git", "commit", "-m", "A"}, env); code != ExitFail || len(sim.calls) != 0 {
		t.Errorf("project fail: %d %v", code, sim.calls)
	}
}

func TestGitPush(t *testing.T) {
	s := &gitSim{branch: "feature/x", top: t.TempDir(), upstream: "origin/main"}
	if code, _, _ := gitRun(t, s, "push"); code != ExitOK || s.calls[0] != "git push -u origin feature/x" {
		t.Errorf("new upstream: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir(), upstream: "origin/feature/x"}
	if code, _, _ := gitRun(t, s, "push"); code != ExitOK || s.calls[0] != "git push" {
		t.Errorf("tracked: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "push", "origin", "feature/x"); code != ExitOK || s.calls[0] != "git push origin feature/x" {
		t.Errorf("explicit: %v", s.calls)
	}
	for _, a := range [][]string{
		{"push", "--force"}, {"push", "-f"}, {"push", "-uf", "origin"}, {"push", "--force-with-lease"},
		{"push", "--force-if-includes"}, {"push", "origin", "+feature/x"}, {"push", "--no-verify"},
	} {
		s := &gitSim{branch: "feature/x", top: t.TempDir()}
		if code, _, _ := gitRun(t, s, a...); code != ExitUsage || len(s.calls) != 0 {
			t.Errorf("%v: %d %v", a, code, s.calls)
		}
	}
}

func TestGitMergeRebase(t *testing.T) {
	s := &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "merge", "origin/main"); code != ExitOK || s.calls[0] != "git merge origin/main" {
		t.Errorf("merge: %v", s.calls)
	}
	s = &gitSim{branch: "main", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "merge", "feature/x"); code != ExitFail || len(s.calls) != 0 {
		t.Errorf("merge on main: %d %v", code, s.calls)
	}
	s = &gitSim{branch: "main", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "merge", "--abort"); code != ExitOK || s.calls[0] != "git merge --abort" {
		t.Errorf("abort: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "merge", "-m", "Co-Authored-By: Claude <noreply@anthropic.com>", "origin/main"); code != ExitLint {
		t.Errorf("merge credit: %d", code)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "merge"); code != ExitUsage {
		t.Errorf("merge bare: %d", code)
	}

	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "rebase", "origin/main"); code != ExitOK || s.calls[0] != "git rebase origin/main" {
		t.Errorf("rebase: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir(), held: true}
	if code, _, errb := gitRun(t, s, "rebase", "origin/main"); code != ExitFail || len(s.calls) != 0 || !strings.Contains(errb, "origin holds") {
		t.Errorf("rebase held: %d %v %q", code, s.calls, errb)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir(), held: true}
	if code, _, _ := gitRun(t, s, "rebase", "--continue"); code != ExitOK || s.calls[0] != "git rebase --continue" {
		t.Errorf("continue: %v", s.calls)
	}
	for _, a := range [][]string{{"rebase", "-i", "origin/main"}, {"rebase", "--interactive", "x"}, {"rebase"}, {"rebase", "--no-verify", "x"}} {
		s := &gitSim{branch: "feature/x", top: t.TempDir()}
		if code, _, _ := gitRun(t, s, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	s = &gitSim{branch: "main", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "rebase", "x"); code != ExitFail {
		t.Errorf("rebase main: %d", code)
	}
}

func TestGitBranch(t *testing.T) {
	s := &gitSim{branch: "feature/x", top: t.TempDir(), hasMain: true}
	if code, _, _ := gitRun(t, s, "branch", "create", "fix/thing"); code != ExitOK || s.calls[0] != "git switch -c fix/thing origin/main" {
		t.Errorf("create: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "branch", "create", "fix/thing", "abc123"); code != ExitOK || s.calls[0] != "git switch -c fix/thing abc123" {
		t.Errorf("create start: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "branch", "create", "fix/thing"); code != ExitOK || s.calls[0] != "git switch -c fix/thing" {
		t.Errorf("create no main: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "branch", "switch", "fix/y"); code != ExitOK || s.calls[0] != "git switch fix/y" {
		t.Errorf("switch: %v", s.calls)
	}
	if code, _, _ := gitRun(t, s, "branch", "list"); code != ExitOK || s.calls[1] != "git branch -vv" {
		t.Errorf("list: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "branch", "rename", "fix/y"); code != ExitOK || s.calls[0] != "git branch -m feature/x fix/y" {
		t.Errorf("rename: %v", s.calls)
	}
	if code, _, _ := gitRun(t, s, "branch", "rename", "fix/a", "fix/b"); code != ExitOK || s.calls[1] != "git branch -m fix/a fix/b" {
		t.Errorf("rename two: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "branch", "delete", "fix/y", "--force", "--remote"); code != ExitOK || strings.Join(s.calls, "|") != "git branch -D fix/y|git push origin --delete fix/y" {
		t.Errorf("delete: %v", s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir(), fail: map[string]int{"git branch -d": 1}}
	if code, _, _ := gitRun(t, s, "branch", "delete", "fix/y", "--remote"); code != 1 || len(s.calls) != 1 {
		t.Errorf("delete unmerged: %d %v", code, s.calls)
	}
	for _, a := range [][]string{
		{"branch"}, {"branch", "create"}, {"branch", "create", "wip"}, {"branch", "create", "-x"}, {"branch", "create", "fix/a", "--force"},
		{"branch", "switch"}, {"branch", "switch", "-x"}, {"branch", "list", "x"}, {"branch", "rename"}, {"branch", "rename", "wip"},
		{"branch", "rename", "--remote", "fix/a"}, {"branch", "delete"}, {"branch", "delete", "main"}, {"branch", "delete", "feature/x"},
		{"branch", "rename", "main", "fix/a"}, {"branch", "weird"},
	} {
		s := &gitSim{branch: "feature/x", top: t.TempDir()}
		if code, _, _ := gitRun(t, s, a...); (code != ExitUsage && code != ExitFail) || len(s.calls) != 0 {
			t.Errorf("%v: %d %v", a, code, s.calls)
		}
	}
}

func TestGitUnknownAndNoExec(t *testing.T) {
	var out, errb bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errb, Getenv: func(string) string { return "" }}
	if code := Run(context.Background(), []string{"git", "commit", "-m", "A"}, env); code != ExitUnavailable {
		t.Errorf("no exec: %d", code)
	}
	if code := Run(context.Background(), []string{"git", "bogus"}, env); code != ExitUsage {
		t.Errorf("unknown: %d", code)
	}
}

// scriptSim makes a project with the scripts that the test names. Each script exits as scripts says.
func scriptSim(t *testing.T, scripts map[string][]int) *gitSim {
	t.Helper()
	top := t.TempDir()
	os.MkdirAll(filepath.Join(top, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(top, "scripts"), 0o755)
	for name := range scripts {
		dir := "scripts"
		if name == "check-wiring.sh" {
			dir = ".claude"
		}
		os.WriteFile(filepath.Join(top, dir, name), []byte("#!/bin/sh\n"), 0o755)
	}
	return &gitSim{branch: "feature/x", top: top, scripts: scripts}
}

func TestGitDashC(t *testing.T) {
	s := &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "-C", "/work/repo", "commit", "-m", "A"); code != ExitOK || s.dir != "/work/repo" || s.calls[0] != "git commit -m A" {
		t.Errorf("commit: %d dir=%q %v", code, s.dir, s.calls)
	}
	s = &gitSim{branch: "feature/x", top: t.TempDir()}
	if code, _, _ := gitRun(t, s, "-C", "/work/repo", "push"); code != ExitOK || s.dir != "/work/repo" {
		t.Errorf("push: %d dir=%q", code, s.dir)
	}
	for _, a := range [][]string{{"-C"}, {"-C", ""}} {
		s = &gitSim{branch: "feature/x", top: t.TempDir()}
		if code, _, _ := gitRun(t, s, append(a, "commit", "-m", "A")...); code != ExitUsage || len(s.calls) != 0 {
			t.Errorf("%v: %d", a, code)
		}
	}
}

func TestGitBranchNameScript(t *testing.T) {
	s := scriptSim(t, map[string][]int{"lint-branch-name.sh": {1}})
	if code, _, errb := gitRun(t, s, "commit", "-m", "A"); code != ExitFail || len(s.calls) != 0 || !strings.Contains(errb, "lint-branch-name.sh") {
		t.Errorf("commit: %d %v %q", code, s.calls, errb)
	}
	s = scriptSim(t, map[string][]int{"lint-branch-name.sh": {1}})
	if code, _, _ := gitRun(t, s, "branch", "create", "feature/ok-name"); code != ExitFail || len(s.calls) != 0 || s.ran[0] != "lint-branch-name.sh feature/ok-name" {
		t.Errorf("create: %d %v %v", code, s.calls, s.ran)
	}
	s = scriptSim(t, map[string][]int{"lint-branch-name.sh": {1}})
	if code, _, _ := gitRun(t, s, "branch", "rename", "feature/new-name"); code != ExitFail || len(s.calls) != 0 {
		t.Errorf("rename: %d %v", code, s.calls)
	}
	s = scriptSim(t, map[string][]int{"lint-branch-name.sh": {0}})
	if code, _, _ := gitRun(t, s, "commit", "-m", "A"); code != ExitOK || s.ran[0] != "lint-branch-name.sh feature/x" {
		t.Errorf("pass: %d %v", code, s.ran)
	}
}

func TestGitMergeWiringRetry(t *testing.T) {
	// The wiring fails, then passes after the merge. The merge stays.
	s := scriptSim(t, map[string][]int{"check-wiring.sh": {1, 0}})
	code, out, errb := gitRun(t, s, "merge", "origin/main")
	if code != ExitOK || s.calls[0] != "git merge origin/main" || len(s.calls) != 1 || !strings.Contains(out, "wiring (after the merge): ok") || !strings.Contains(errb, "runs the merge") {
		t.Errorf("heals: %d %v %q %q", code, s.calls, out, errb)
	}
	// The wiring still fails after the merge. The branch goes back.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	if code, _, errb := gitRun(t, s, "merge", "origin/main"); code != ExitFail || len(s.calls) != 2 || s.calls[1] != "git reset -q --hard abc123" || !strings.Contains(errb, "still fails") {
		t.Errorf("still fails: %d %v %q", code, s.calls, errb)
	}
	// A dirty tree can't take the trial merge.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	s.dirty = true
	if code, _, _ := gitRun(t, s, "merge", "origin/main"); code != ExitFail || len(s.calls) != 0 {
		t.Errorf("dirty: %d %v", code, s.calls)
	}
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	s.merging = true
	if code, _, _ := gitRun(t, s, "merge", "origin/main"); code != ExitFail || len(s.calls) != 0 {
		t.Errorf("in merge: %d %v", code, s.calls)
	}
	// A failed merge keeps its exit code, and nothing resets.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	s.fail = map[string]int{"git merge": 1}
	if code, _, _ := gitRun(t, s, "merge", "origin/main"); code != ExitFail || len(s.calls) != 1 {
		t.Errorf("conflict: %d %v", code, s.calls)
	}
	// A good wiring check merges once, with no retry.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {0}})
	if code, out, _ := gitRun(t, s, "merge", "origin/main"); code != ExitOK || strings.Contains(out, "after the merge") || len(s.ran) != 1 {
		t.Errorf("good: %d %q %v", code, out, s.ran)
	}
	// --continue runs the wiring check, and refuses when it fails.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	if code, _, _ := gitRun(t, s, "merge", "--continue"); code != ExitFail || len(s.calls) != 0 {
		t.Errorf("continue fail: %d %v", code, s.calls)
	}
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {0}})
	if code, _, _ := gitRun(t, s, "merge", "--continue"); code != ExitOK || s.calls[0] != "git merge --continue" {
		t.Errorf("continue: %d %v", code, s.calls)
	}
	// --abort skips every check.
	s = scriptSim(t, map[string][]int{"check-wiring.sh": {1}})
	if code, _, _ := gitRun(t, s, "merge", "--abort"); code != ExitOK || len(s.ran) != 0 {
		t.Errorf("abort: %d %v", code, s.ran)
	}
}
