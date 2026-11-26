package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// glob matches a check context against a required pattern. A star matches any run of
// characters, the same way the branch protection rule of the server reads it.
func glob(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for _, p := range parts[1 : len(parts)-1] {
		i := strings.Index(s, p)
		if i < 0 {
			return false
		}
		s = s[i+len(p):]
	}
	return strings.HasSuffix(s, parts[len(parts)-1])
}

// watchState reads one poll. A driver without PRWatcher gets a state that PRStatus builds. That
// state has one check for the whole CI summary, and the conflict list of the server.
func watchState(ctx context.Context, c forge.Client, n int) (ws forge.WatchState, conflicts []string, serverConflicts bool, err error) {
	if w, ok := c.(forge.PRWatcher); ok {
		ws, err = w.PRWatch(ctx, n)
		return ws, nil, false, err
	}
	s, err := c.PRStatus(ctx, n)
	if err != nil {
		return ws, nil, false, err
	}
	ws = forge.WatchState{State: s.State, HeadSHA: s.HeadSHA}
	switch s.CI {
	case "none":
	case "failure":
		ws.Checks = []forge.Check{{Context: "ci", Status: "failure"}}
	case "pending":
		ws.Checks = []forge.Check{{Context: "ci", Status: "pending"}}
	default:
		ws.Checks = []forge.Check{{Context: "ci", Status: "success"}}
	}
	return ws, s.ConflictedFiles, true, nil
}

// gitOut runs git through Exec and returns the output and the exit code.
func gitOut(ctx context.Context, env Env, args ...string) (string, int) {
	var out, errb bytes.Buffer
	code := env.Exec(ctx, "git", args, &out, &errb)
	return out.String(), code
}

// mergeConflicts fetches the base branch and the head of the pull request, then runs
// `git merge-tree` on them. It doesn't trust the conflict list of the server: some servers
// return null for a real conflict. It returns the conflicted files, none for a clean merge.
func mergeConflicts(ctx context.Context, env Env, remote, base string, n int) ([]string, error) {
	baseRef := fmt.Sprintf("refs/watch-pr/%d/base", n)
	headRef := fmt.Sprintf("refs/watch-pr/%d/head", n)
	defer func() {
		gitOut(ctx, env, "update-ref", "-d", baseRef)
		gitOut(ctx, env, "update-ref", "-d", headRef)
	}()
	// An empty --refmap stops git from also updating refs/remotes/<remote>/<base>. Two watchers
	// at once, or a git fetch in another shell, then can't fight over the lock of that ref.
	if _, code := gitOut(ctx, env, "fetch", "-q", "--refmap=", "--no-write-fetch-head", remote,
		"+refs/heads/"+base+":"+baseRef, fmt.Sprintf("+refs/pull/%d/head:%s", n, headRef)); code != 0 {
		return nil, fmt.Errorf("git fetch of %s failed (exit %d)", remote, code)
	}
	out, code := gitOut(ctx, env, "merge-tree", "--write-tree", "--name-only", "--no-messages", baseRef, headRef)
	switch code {
	case 0:
		return nil, nil
	case 1:
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		return lines[1:], nil
	}
	return nil, fmt.Errorf("git merge-tree failed (exit %d)", code)
}

// foreignFolder reports whether --host and --repo name a repo other than the one the remote of
// the working folder points to. A folder with no readable remote counts as foreign too.
func foreignFolder(ctx context.Context, env Env, remote string, o opts) bool {
	if o.host == "" && o.repo == "" {
		return false
	}
	out, code := gitOut(ctx, env, "remote", "get-url", remote)
	if code != 0 {
		return true
	}
	h, r, err := forge.ParseRemote(strings.TrimSpace(out))
	if err != nil {
		return true
	}
	return (o.host != "" && !strings.EqualFold(o.host, h)) || (o.repo != "" && !strings.EqualFold(o.repo, r))
}

// pidFile gives the path of the pid file of one watch. It lives in the shared git dir, so every
// worktree of the repository sees it.
func pidFile(ctx context.Context, env Env, n int) (string, error) {
	if env.Exec == nil {
		return "", fmt.Errorf("this build can't run git")
	}
	out, code := gitOut(ctx, env, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if code != 0 {
		return "", fmt.Errorf("this folder is not a git repository")
	}
	return filepath.Join(strings.TrimSpace(out), fmt.Sprintf("watch-pr-%d.pid", n)), nil
}

func pidAlive(env Env, pid int) bool {
	if env.PidAlive != nil {
		return env.PidAlive(pid)
	}
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}

func ownPid(env Env) int {
	if env.Pid != nil {
		return env.Pid()
	}
	return os.Getpid()
}

// livePid returns the pid in the file when a process holds it.
func livePid(env Env, path string) (int, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid < 1 || !pidAlive(env, pid) {
		return 0, false
	}
	return pid, true
}

// watch polls the pull request until it settles or the timeout ends. It prints one line each
// time the state changes, and one RESULT line at the end.
func watch(ctx context.Context, c forge.Client, env Env, n int, o opts) int {
	if env.Exec == nil {
		fmt.Fprintln(env.Stderr, "vcs: this build can't run git")
		return ExitUnavailable
	}
	path, err := pidFile(ctx, env, n)
	if err != nil {
		fmt.Fprintf(env.Stderr, "vcs: pr watch: %v\n", err)
		return ExitFail
	}
	if o.flags["status"] == "true" {
		if pid, ok := livePid(env, path); ok {
			fmt.Fprintf(env.Stdout, "watching #%d pid=%d\n", n, pid)
			return ExitOK
		}
		fmt.Fprintf(env.Stdout, "not watching #%d\n", n)
		return ExitFail
	}
	self := ownPid(env)
	if pid, ok := livePid(env, path); ok && pid != self {
		fmt.Fprintf(env.Stderr, "vcs: pr watch: #%d already has a watcher, pid %d\n", n, pid)
		return ExitBusy
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(self)+"\n"), 0o644); err != nil {
		fmt.Fprintf(env.Stderr, "vcs: pr watch: %v\n", err)
		return ExitFail
	}
	defer func() {
		// Remove the file only when it still holds this watcher's pid.
		if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) == strconv.Itoa(self) {
			os.Remove(path)
		}
	}()
	return poll(ctx, c, env, n, o)
}

func poll(ctx context.Context, c forge.Client, env Env, n int, o opts) int {
	interval, _ := intFlag(o, "interval", 30)
	timeout, _ := intFlag(o, "timeout", 3600)
	remote := o.flags["remote"]
	if remote == "" {
		remote = "origin"
	}
	foreign := foreignFolder(ctx, env, remote, o)
	sleep := env.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) {
			select {
			case <-ctx.Done():
			case <-time.After(d):
			}
		}
	}
	now := time.Now
	if env.Now != nil {
		now = env.Now
	}
	result := func(code int, text string) int {
		fmt.Fprintf(env.Stdout, "RESULT: %s\n", text)
		return code
	}
	waited := 0
	last := ""
	for {
		ws, conflicted, fromServer, err := watchState(ctx, c, n)
		if err != nil {
			return fail(env, err)
		}
		if ws.State != "open" {
			return result(ExitClosed, "the pull request is "+ws.State)
		}
		if !fromServer {
			if foreign {
				// The folder belongs to another repo, so a fetch can't reach the pull request.
				// The conflict list of the server stands in for the local merge check.
				s, err := c.PRStatus(ctx, n)
				if err != nil {
					return fail(env, err)
				}
				conflicted = s.ConflictedFiles
			} else if conflicted, err = mergeConflicts(ctx, env, remote, ws.Base, n); err != nil {
				fmt.Fprintf(env.Stderr, "vcs: pr watch: %v\n", err)
				return ExitFail
			}
		}

		pending, failed, missing := 0, []forge.Check{}, []string{}
		for _, ck := range ws.Checks {
			switch ck.Status {
			case "pending":
				pending++
			case "failure", "error":
				failed = append(failed, ck)
			}
		}
		for _, pattern := range ws.Required {
			found := false
			for _, ck := range ws.Checks {
				if glob(pattern, ck.Context) {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, pattern)
			}
		}
		total := len(ws.Checks)
		ci := "green"
		switch {
		case len(failed) > 0:
			ci = "failed"
		case total == 0 || pending > 0 || len(missing) > 0:
			ci = "pending"
		}
		merge := "clean"
		if len(conflicted) > 0 {
			merge = "conflict"
		}
		sha := ws.HeadSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		line := fmt.Sprintf("head=%s ci=%s (%d/%d done, %d failed) merge=%s", sha, ci, total-pending, total, len(failed), merge)
		if line != last {
			fmt.Fprintf(env.Stdout, "%s #%d %s\n", now().UTC().Format("2006-01-02T15:04:05Z"), n, line)
			last = line
		}

		if len(conflicted) > 0 {
			fmt.Fprintln(env.Stdout, "Conflicted files:")
			for _, f := range conflicted {
				fmt.Fprintf(env.Stdout, "  %s\n", f)
			}
			return result(ExitConflict, "the pull request conflicts with "+ws.Base)
		}
		if len(failed) > 0 {
			fmt.Fprintln(env.Stdout, "Failed checks:")
			for _, ck := range failed {
				fmt.Fprintf(env.Stdout, "  %s  %s\n", ck.Context, ck.URL)
			}
			return result(ExitFail, "a check failed")
		}
		if ci == "green" {
			return result(ExitOK, "every check passed and the pull request merges cleanly")
		}
		if waited >= timeout || ctx.Err() != nil {
			if len(missing) > 0 {
				fmt.Fprintln(env.Stdout, "Required checks with no status:")
				for _, m := range missing {
					fmt.Fprintf(env.Stdout, "  %s\n", m)
				}
			}
			return result(ExitTimeout, fmt.Sprintf("timeout after %ds", timeout))
		}
		sleep(ctx, time.Duration(interval)*time.Second)
		waited += interval
	}
}
