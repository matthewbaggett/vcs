package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// wfake returns each state in turn, then repeats the last one. It is a PRWatcher.
type wfake struct {
	fake
	states []forge.WatchState
	calls  *int
}

func (w wfake) PRWatch(context.Context, int) (forge.WatchState, error) {
	i := *w.calls
	*w.calls++
	if i >= len(w.states) {
		i = len(w.states) - 1
	}
	return w.states[i], nil
}

// sfake has no PRWatcher, so the command builds the state from PRStatus.
type sfake struct {
	fake
	status forge.PRStatus
}

func (s sfake) PRStatus(context.Context, int) (forge.PRStatus, error) { return s.status, nil }

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// watchRepo builds a remote with main and a pull request head, clones it, and moves into the
// clone. With conflict, both sides edit one line.
func watchRepo(t *testing.T, conflict bool) string {
	t.Helper()
	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	gitIn(t, root, "init", "-q", "-b", "main", origin)
	os.WriteFile(filepath.Join(origin, "file.txt"), []byte("base\n"), 0o644)
	gitIn(t, origin, "add", "file.txt")
	gitIn(t, origin, "commit", "-qm", "base")
	gitIn(t, origin, "switch", "-qc", "topic")
	name := "other.txt"
	if conflict {
		name = "file.txt"
	}
	os.WriteFile(filepath.Join(origin, name), []byte("topic\n"), 0o644)
	gitIn(t, origin, "add", "-A")
	gitIn(t, origin, "commit", "-qm", "topic")
	gitIn(t, origin, "update-ref", "refs/pull/7/head", "topic")
	gitIn(t, origin, "switch", "-q", "main")
	if conflict {
		os.WriteFile(filepath.Join(origin, "file.txt"), []byte("main\n"), 0o644)
		gitIn(t, origin, "commit", "-qam", "main")
	}
	clone := filepath.Join(root, "repo")
	gitIn(t, root, "clone", "-q", origin, clone)
	t.Chdir(clone)
	return clone
}

func realExec(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

// wrun runs `vcs pr watch 7` in the current folder with the given client.
func wrun(t *testing.T, c forge.Client, args ...string) (code int, out, errs string, sleeps int) {
	t.Helper()
	var o, e bytes.Buffer
	env := Env{
		Stdout: &o, Stderr: &e, IsTerminal: true,
		Getenv:    func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return c, nil },
		Sleep:     func(context.Context, time.Duration) { sleeps++ },
		Now:       func() time.Time { return time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC) },
		Exec:      realExec,
	}
	code = Run(context.Background(), append([]string{"pr", "watch", "7"}, args...), env)
	return code, o.String(), e.String(), sleeps
}

func states(ws ...forge.WatchState) forge.Client {
	calls := 0
	return wfake{states: ws, calls: &calls}
}

func openState(checks ...forge.Check) forge.WatchState {
	return forge.WatchState{State: "open", HeadSHA: "abc1234def", Base: "main", Checks: checks}
}

func ck(name, status string) forge.Check {
	return forge.Check{Context: name, Status: status, URL: "/run/1"}
}

func TestWatchGreen(t *testing.T) {
	watchRepo(t, false)
	code, out, _, _ := wrun(t, states(openState(ck("CI / a (pull_request)", "success"), ck("CI / b (pull_request)", "success"))))
	if code != ExitOK {
		t.Fatalf("code=%d out=%s", code, out)
	}
	want := "2026-10-17T10:00:00Z #7 head=abc1234 ci=green (2/2 done, 0 failed) merge=clean\n" +
		"RESULT: every check passed and the pull request merges cleanly\n"
	if out != want {
		t.Fatalf("out=%q", out)
	}
}

func TestWatchWaitsThenGreen(t *testing.T) {
	watchRepo(t, false)
	p := openState(ck("a", "success"), ck("b", "pending"))
	g := openState(ck("a", "success"), ck("b", "success"))
	code, out, _, sleeps := wrun(t, states(p, p, g))
	if code != ExitOK || sleeps != 2 {
		t.Fatalf("code=%d sleeps=%d", code, sleeps)
	}
	// One line for each change of state: the repeated pending poll prints nothing.
	if n := strings.Count(out, " #7 head="); n != 2 {
		t.Fatalf("state lines = %d\n%s", n, out)
	}
	if !strings.Contains(out, "ci=pending (1/2 done, 0 failed) merge=clean") {
		t.Fatalf("out=%s", out)
	}
}

func TestWatchFailedCheck(t *testing.T) {
	watchRepo(t, false)
	for _, status := range []string{"failure", "error"} {
		code, out, _, _ := wrun(t, states(openState(ck("CI / a (pull_request)", "success"), ck("CI / b (pull_request)", status))))
		if code != ExitFail {
			t.Fatalf("%s: code=%d", status, code)
		}
		for _, want := range []string{"ci=failed (2/2 done, 1 failed)", "Failed checks:\n  CI / b (pull_request)  /run/1\n", "RESULT: a check failed\n"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in\n%s", status, want, out)
			}
		}
	}
}

func TestWatchConflict(t *testing.T) {
	watchRepo(t, true)
	// A conflict ends the watch even while a check is pending.
	code, out, _, _ := wrun(t, states(openState(ck("CI / a (pull_request)", "pending"))))
	if code != ExitConflict {
		t.Fatalf("code=%d out=%s", code, out)
	}
	for _, want := range []string{"merge=conflict", "Conflicted files:\n  file.txt\n", "RESULT: the pull request conflicts with main\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestWatchCleanMergeLeavesNoRefs(t *testing.T) {
	clone := watchRepo(t, false)
	wrun(t, states(openState(ck("a", "success"))))
	if refs := gitIn(t, clone, "for-each-ref", "refs/watch-pr"); refs != "" {
		t.Fatalf("refs left behind: %s", refs)
	}
}

func TestWatchClosedAndMerged(t *testing.T) {
	watchRepo(t, false)
	for _, state := range []string{"closed", "merged"} {
		code, out, _, _ := wrun(t, states(forge.WatchState{State: state}))
		if code != ExitClosed || out != "RESULT: the pull request is "+state+"\n" {
			t.Errorf("%s: code=%d out=%q", state, code, out)
		}
	}
}

func TestWatchTimeout(t *testing.T) {
	watchRepo(t, false)
	pending := openState(ck("a", "pending"))
	code, out, _, sleeps := wrun(t, states(pending), "--interval", "10", "--timeout", "30")
	if code != ExitTimeout || sleeps != 3 {
		t.Fatalf("code=%d sleeps=%d", code, sleeps)
	}
	if !strings.Contains(out, "ci=pending (0/1 done, 0 failed)") || !strings.HasSuffix(out, "RESULT: timeout after 30s\n") {
		t.Fatalf("out=%s", out)
	}
}

func TestWatchNoStatusesIsPending(t *testing.T) {
	watchRepo(t, false)
	code, out, _, _ := wrun(t, states(openState()), "--interval", "1", "--timeout", "1")
	if code != ExitTimeout || !strings.Contains(out, "ci=pending (0/0 done") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestWatchRequiredChecks(t *testing.T) {
	watchRepo(t, false)
	required := []string{"CI / * (pull_request)", "Lint / * (pull_request_target)"}
	with := func(checks ...forge.Check) forge.Client {
		ws := openState(checks...)
		ws.Required = required
		return states(ws)
	}
	// A required check with no status counts as pending, and the timeout names it.
	code, out, _, _ := wrun(t, with(ck("CI / a (pull_request)", "success")), "--interval", "1", "--timeout", "1")
	if code != ExitTimeout || !strings.Contains(out, "Required checks with no status:\n  Lint / * (pull_request_target)\n") {
		t.Fatalf("missing: code=%d out=%s", code, out)
	}
	// Every required check present: green.
	code, _, _, _ = wrun(t, with(ck("CI / a (pull_request)", "success"), ck("Lint / x (pull_request_target)", "success")))
	if code != ExitOK {
		t.Fatalf("present: code=%d", code)
	}
	// A push status doesn't satisfy a pull_request rule.
	code, out, _, _ = wrun(t, with(ck("CI / a (push)", "success"), ck("Lint / x (pull_request_target)", "success")), "--interval", "1", "--timeout", "1")
	if code != ExitTimeout || !strings.Contains(out, "  CI / * (pull_request)\n") {
		t.Fatalf("push status: code=%d out=%s", code, out)
	}
}

func TestGlob(t *testing.T) {
	for _, c := range []struct {
		p, s string
		want bool
	}{
		{"a", "a", true}, {"a", "b", false}, {"CI / *", "CI / x (push)", true}, {"*", "", true},
		{"a*c", "abc", true}, {"a*c", "ab", false}, {"a*b*c", "axbyc", true}, {"a*b*c", "acb", false},
	} {
		if got := glob(c.p, c.s); got != c.want {
			t.Errorf("glob(%q, %q) = %v", c.p, c.s, got)
		}
	}
}

// A driver without PRWatcher: the state comes from PRStatus, and the conflict list from the server.
func TestWatchFallbackFromPRStatus(t *testing.T) {
	watchRepo(t, false)
	code, out, _, _ := wrun(t, sfake{status: forge.PRStatus{State: "open", CI: "success", HeadSHA: "abc1234def"}})
	if code != ExitOK || !strings.Contains(out, "ci=green (1/1 done, 0 failed)") {
		t.Fatalf("green: code=%d out=%s", code, out)
	}
	code, _, _, _ = wrun(t, sfake{status: forge.PRStatus{State: "open", CI: "failure"}})
	if code != ExitFail {
		t.Fatalf("failure: code=%d", code)
	}
	code, out, _, _ = wrun(t, sfake{status: forge.PRStatus{State: "open", CI: "pending", ConflictedFiles: []string{"a.go"}}})
	if code != ExitConflict || !strings.Contains(out, "  a.go\n") {
		t.Fatalf("conflict: code=%d out=%s", code, out)
	}
	code, _, _, _ = wrun(t, sfake{status: forge.PRStatus{State: "merged"}})
	if code != ExitClosed {
		t.Fatalf("merged: code=%d", code)
	}
	code, out, _, _ = wrun(t, sfake{status: forge.PRStatus{State: "open", CI: "none"}}, "--interval", "1", "--timeout", "1")
	if code != ExitTimeout || !strings.Contains(out, "ci=pending (0/0 done") {
		t.Fatalf("none: code=%d out=%s", code, out)
	}
}

func TestWatchPidFile(t *testing.T) {
	clone := watchRepo(t, false)
	pidPath := filepath.Join(clone, ".git", "watch-pr-7.pid")
	g := states(openState(ck("a", "success")))

	// A finished watch removes its pid file.
	if code, _, _, _ := wrun(t, g); code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if _, err := os.Stat(pidPath); err == nil {
		t.Fatal("pid file stays after a finished watch")
	}

	// A stale file, left by a killed watcher: --status says none, and a new watch replaces the file.
	os.WriteFile(pidPath, []byte("999999\n"), 0o644)
	code, out, _, _ := wrun(t, g, "--status")
	if code != ExitFail || out != "not watching #7\n" {
		t.Fatalf("dead pid: code=%d out=%q", code, out)
	}
	if code, _, _, _ := wrun(t, g); code != ExitOK {
		t.Fatalf("start over a stale pid: code=%d", code)
	}
	if _, err := os.Stat(pidPath); err == nil {
		t.Fatal("stale file stays")
	}

	// A live pid: --status says watching, and a second watcher refuses with exit 5.
	live := os.Getpid()
	os.WriteFile(pidPath, []byte(strconv.Itoa(live)+"\n"), 0o644)
	code, out, _, _ = wrun(t, g, "--status")
	if code != ExitOK || out != "watching #7 pid="+strconv.Itoa(live)+"\n" {
		t.Fatalf("live pid: code=%d out=%q", code, out)
	}
	var o, e bytes.Buffer
	env := Env{Stdout: &o, Stderr: &e, Getenv: func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return g, nil },
		Exec:      realExec, Pid: func() int { return live + 1 }, PidAlive: func(p int) bool { return p == live }}
	if code := Run(context.Background(), []string{"pr", "watch", "7"}, env); code != ExitBusy {
		t.Fatalf("second watcher: code=%d", code)
	}
	if !strings.Contains(e.String(), "already has a watcher, pid "+strconv.Itoa(live)) {
		t.Fatalf("stderr=%q", e.String())
	}
	if raw, _ := os.ReadFile(pidPath); strings.TrimSpace(string(raw)) != strconv.Itoa(live) {
		t.Fatalf("the live pid file changed: %q", raw)
	}
}

func TestWatchUsage(t *testing.T) {
	watchRepo(t, false)
	g := states(openState(ck("a", "success")))
	for _, a := range [][]string{{"--interval", "0"}, {"--timeout", "x"}, {"--bogus"}} {
		if code, _, _, _ := wrun(t, g, a...); code != ExitUsage {
			t.Errorf("%v: code=%d", a, code)
		}
	}
	var e bytes.Buffer
	env := Env{Stdout: io.Discard, Stderr: &e, Getenv: func(string) string { return "" }}
	for _, a := range [][]string{{"pr", "watch"}, {"pr", "watch", "seven"}, {"pr", "watch", "7", "8"}} {
		if code := Run(context.Background(), a, env); code != ExitUsage {
			t.Errorf("%v: code=%d", a, code)
		}
	}
}

func TestWatchNeedsGit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	code, _, errs, _ := wrun(t, states(openState(ck("a", "success"))))
	if code != ExitFail || !strings.Contains(errs, "not a git repository") {
		t.Fatalf("code=%d err=%s", code, errs)
	}
	var e bytes.Buffer
	env := Env{Stdout: io.Discard, Stderr: &e, Getenv: func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return states(openState()), nil }}
	if code := Run(context.Background(), []string{"pr", "watch", "7"}, env); code != ExitUnavailable {
		t.Fatalf("no exec: code=%d", code)
	}
}

// The --status flag of pr watch takes no value, and --status of ci runs still takes one.
func TestStatusFlagParse(t *testing.T) {
	o, err := parse([]string{"7", "--status"})
	if err != nil || o.flags["status"] != "true" {
		t.Fatalf("bare: %v %v", o.flags, err)
	}
	o, err = parse([]string{"--status", "failure", "--limit", "3"})
	if err != nil || o.flags["status"] != "failure" {
		t.Fatalf("valued: %v %v", o.flags, err)
	}
}

// A merged pull request gets its own hint. It must not tell the agent to mark it ready.
func TestWatchMergedHint(t *testing.T) {
	watchRepo(t, false)
	_, _, errs, _ := wrun(t, states(forge.WatchState{State: "merged"}), "--caller", "agent")
	if !strings.Contains(errs, "has landed or someone closed it") || strings.Contains(errs, "pr ready") {
		t.Fatalf("hint=%q", errs)
	}
}

// --repo names a repo other than the one the folder holds. The fetch of origin would fail, so
// the verb reads the conflict list of the server and never fetches.
func TestWatchForeignRepoSkipsFetch(t *testing.T) {
	clone := watchRepo(t, false)
	gitIn(t, clone, "remote", "set-url", "origin", "/nonexistent/path")
	green := openState(ck("a", "success"))
	code, out, errs, _ := wrun(t, wfakeStatus{wfake: states(green).(wfake), status: forge.PRStatus{}}, "--host", "git.example", "--repo", "o/scratch")
	if code != ExitOK || strings.Contains(errs, "git fetch") {
		t.Fatalf("code=%d out=%s err=%s", code, out, errs)
	}
	conflict := wfakeStatus{wfake: states(green).(wfake), status: forge.PRStatus{ConflictedFiles: []string{"x.txt"}}}
	code, out, _, _ = wrun(t, conflict, "--host", "git.example", "--repo", "o/scratch")
	if code != ExitConflict || !strings.Contains(out, "  x.txt\n") {
		t.Fatalf("conflict: code=%d out=%s", code, out)
	}
}

// wfakeStatus is a PRWatcher with a PRStatus of its own.
type wfakeStatus struct {
	wfake
	status forge.PRStatus
}

func (w wfakeStatus) PRStatus(context.Context, int) (forge.PRStatus, error) { return w.status, nil }
