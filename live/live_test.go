//go:build live

// Package live runs the built binary against a real Forgejo repository.
//
// The unit tests fake the forge, so they can't see how the real server behaves. This suite does.
// It runs only with the `live` build tag and a token. See docs/Live-Tests.md.
package live

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	bin      string
	host     = envOr("VCS_LIVE_HOST", "git.grey.ooo")
	repo     = envOr("VCS_LIVE_REPO", "geusebio/vcs-scratch")
	stamp    = strconv.FormatInt(time.Now().Unix(), 10)
	cloneDir string
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func TestMain(m *testing.M) {
	if os.Getenv("VCS_TOKEN") == "" {
		fmt.Println("live tests need VCS_TOKEN. Skipping.")
		os.Exit(0)
	}
	dir, err := os.MkdirTemp("", "vcs-live")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	bin = filepath.Join(dir, "vcs")
	if out, err := exec.Command("go", "build", "-o", bin, "../cmd/vcs").CombinedOutput(); err != nil {
		fmt.Printf("build: %v\n%s", err, out)
		os.Exit(2)
	}
	cloneDir = filepath.Join(dir, "clone")
	url := "https://oauth2:" + os.Getenv("VCS_TOKEN") + "@" + host + "/" + repo + ".git"
	if out, err := exec.Command("git", "clone", "-q", url, cloneDir).CombinedOutput(); err != nil {
		fmt.Printf("clone: %v\n%s", err, strings.ReplaceAll(string(out), os.Getenv("VCS_TOKEN"), "<token>"))
		os.Exit(2)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// result is one run of the binary.
type result struct {
	out, err string
	code     int
}

// vcs runs the binary in the clone, which has no .vale.ini, so only the lexi and credit checks run.
func vcs(t *testing.T, args ...string) result {
	t.Helper()
	return vcsIn(t, cloneDir, args...)
}

func vcsIn(t *testing.T, dir string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run vcs %v: %v", args, err)
	}
	return result{so.String(), se.String(), code}
}

// target adds the host and the repo to a verb that reads or writes the forge.
func target(args ...string) []string {
	return append(args, "--host", host, "--repo", repo)
}

// ok runs a verb and fails the test unless it exits 0.
func ok(t *testing.T, args ...string) result {
	t.Helper()
	r := vcs(t, target(args...)...)
	if r.code != 0 {
		t.Fatalf("vcs %s: exit %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), r.code, r.out, r.err)
	}
	return r
}

// exits runs a verb and fails the test unless it exits with the wanted code.
func exits(t *testing.T, want int, args ...string) result {
	t.Helper()
	r := vcs(t, target(args...)...)
	if r.code != want {
		t.Fatalf("vcs %s: exit %d, want %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), r.code, want, r.out, r.err)
	}
	return r
}

func rows(s string) [][]string {
	var out [][]string
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if l != "" {
			out = append(out, strings.Split(l, "\t"))
		}
	}
	return out
}

func firstField(t *testing.T, s string) string {
	t.Helper()
	r := rows(s)
	if len(r) == 0 || r[0][0] == "" {
		t.Fatalf("no row in %q", s)
	}
	return r[0][0]
}

// numberOf reads the number from a `<n> <address>` row of a create verb.
func numberOf(t *testing.T, r result) string {
	t.Helper()
	return firstField(t, r.out)
}

func contains(t *testing.T, what, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s: %q does not contain %q", what, got, want)
	}
}

// git runs plain git in the clone for the setup that the vcs verbs don't cover.
func git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cloneDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, name, body string) {
	t.Helper()
	p := filepath.Join(cloneDir, name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// landChange pushes one file on a new branch, opens a pull request, and merges it.
// It returns the pull request number.
func landChange(t *testing.T, branch, file, body string, merge bool) string {
	t.Helper()
	vcs(t, "git", "branch", "switch", "main")
	git(t, "pull", "-q", "--ff-only")
	if r := vcs(t, "git", "branch", "create", branch); r.code != 0 {
		t.Fatalf("branch create: %s", r.err)
	}
	writeFile(t, file, body)
	git(t, "add", file)
	if r := vcs(t, "git", "commit", "-m", "Add "+file); r.code != 0 {
		t.Fatalf("commit: %s", r.err)
	}
	if r := vcs(t, "git", "push", "-u", "origin", branch); r.code != 0 {
		t.Fatalf("push: %s", r.err)
	}
	n := numberOf(t, ok(t, "pr", "create", "Add "+file, "--head", branch, "--body", "A live test change."))
	if merge {
		ok(t, "pr", "merge", n, "--method", "squash", "--delete-branch")
		vcs(t, "git", "branch", "switch", "main")
		git(t, "pull", "-q", "--ff-only")
	}
	return n
}

// poll runs a check each few seconds until it passes or the time ends.
func poll(t *testing.T, limit time.Duration, what string, check func() bool) {
	t.Helper()
	end := time.Now().Add(limit)
	for time.Now().Before(end) {
		if check() {
			return
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("timed out after %s: %s", limit, what)
}

// Every verb prints help. This needs no forge.
func TestHelpOfEveryVerb(t *testing.T) {
	pages, _ := filepath.Glob("../docs/Commands/*.md")
	if len(pages) == 0 {
		t.Skip("no command pages")
	}
	for _, p := range pages {
		name := strings.TrimSuffix(filepath.Base(p), ".md")
		if name == "README" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if r := vcsIn(t, ".", append([]string{"help"}, strings.Split(name, "-")...)...); r.code != 0 {
				t.Errorf("help %s: exit %d: %s", name, r.code, r.err)
			}
		})
	}
}

func TestIssues(t *testing.T) {
	n := numberOf(t, ok(t, "issue", "create", "Live issue "+stamp, "--body", "A live test body."))
	defer vcs(t, target("issue", "edit", n, "--state", "closed")...)

	t.Run("view", func(t *testing.T) {
		r := ok(t, "issue", "view", n)
		contains(t, "view", r.out, "Live issue "+stamp)
		contains(t, "view", r.out, "A live test body.")
	})
	t.Run("edit", func(t *testing.T) {
		ok(t, "issue", "edit", n, "--title", "Live issue renamed "+stamp)
		contains(t, "view", ok(t, "issue", "view", n).out, "renamed")
	})
	t.Run("close and reopen", func(t *testing.T) {
		ok(t, "issue", "edit", n, "--state", "closed")
		contains(t, "closed", ok(t, "issue", "list", "--state", "closed").out, "renamed "+stamp)
		ok(t, "issue", "edit", n, "--state", "open")
		contains(t, "open", ok(t, "issue", "list").out, "renamed "+stamp)
	})
	t.Run("top and newer", func(t *testing.T) {
		if firstField(t, ok(t, "issue", "top").out) == "" {
			t.Error("no top")
		}
		contains(t, "newer", ok(t, "issue", "newer", "0").out, n)
	})
	t.Run("timeline", func(t *testing.T) {
		ok(t, "issue", "timeline", n)
	})
	t.Run("comments", func(t *testing.T) {
		add := ok(t, "comment", "add", n, "--body", "A live comment.")
		id := firstField(t, add.out)
		contains(t, "list", ok(t, "comment", "list", n).out, "A live comment.")
		ok(t, "comment", "edit", id, "--body", "An edited live comment.")
		contains(t, "edited", ok(t, "comment", "list", n).out, "An edited live comment.")
		ok(t, "comment", "delete", id)
		if strings.Contains(ok(t, "comment", "list", n).out, "edited live comment") {
			t.Error("the comment is still there")
		}
	})
	t.Run("time", func(t *testing.T) {
		ok(t, "time", "add", n, "90m")
		ok(t, "time", "list", n)
		contains(t, "total", ok(t, "time", "total", n).out, "5400")
	})
	t.Run("attach", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "live.txt")
		os.WriteFile(f, []byte("live"), 0o644)
		contains(t, "attach", ok(t, "attach", n, f).out, "live.txt")
	})
	t.Run("milestone and labels", func(t *testing.T) {
		ms, lb := "live-ms-"+stamp, "live-lb-"+stamp
		ok(t, "milestone", "create", ms, "--desc", "A live milestone.")
		contains(t, "milestone list", ok(t, "milestone", "list").out, ms)
		contains(t, "milestone view", ok(t, "milestone", "view", ms).out, ms)
		ok(t, "label", "create", lb, "00ff00")
		var id string
		for _, r := range rows(ok(t, "label", "list").out) {
			if len(r) > 1 && r[1] == lb {
				id = r[0]
			}
		}
		if id == "" {
			t.Fatal("the label is not in the list")
		}
		ok(t, "label", "rename", id, lb+"-b")
		contains(t, "renamed", ok(t, "label", "list").out, lb+"-b")
		ok(t, "issue", "edit", n, "--milestone", ms, "--add-label", lb+"-b")
		r := ok(t, "issue", "view", n).out
		contains(t, "milestone on issue", r, ms)
		contains(t, "label on issue", r, lb+"-b")
		ok(t, "issue", "edit", n, "--milestone", "none", "--rm-label", lb+"-b")
		if strings.Contains(ok(t, "issue", "view", n).out, ms) {
			t.Error("the milestone is still set")
		}
	})
}

// The checks inside the binary stop a bad text before it leaves.
func TestGuards(t *testing.T) {
	t.Run("insensitive words", func(t *testing.T) {
		r := exits(t, 65, "issue", "create", "Words "+stamp, "--body", "This is a stupid hack.")
		contains(t, "stderr", r.err, "stupid")
	})
	t.Run("credit line", func(t *testing.T) {
		r := exits(t, 65, "issue", "create", "Credit "+stamp, "--body", "Generated with Claude Code.")
		contains(t, "stderr", r.err, "credit")
	})
	t.Run("usage error", func(t *testing.T) {
		exits(t, 64, "issue", "view")
	})
	t.Run("not found", func(t *testing.T) {
		exits(t, 1, "issue", "view", "999999")
	})
}

func TestGitAndPullRequests(t *testing.T) {
	t.Run("no commit on main", func(t *testing.T) {
		vcs(t, "git", "branch", "switch", "main")
		writeFile(t, "never.txt", "never")
		git(t, "add", "never.txt")
		if r := vcs(t, "git", "commit", "-m", "Add never"); r.code == 0 {
			t.Error("a commit on main went through")
		}
		git(t, "reset", "-q", "--hard")
	})
	branch := "fix/live-" + stamp
	vcs(t, "git", "branch", "switch", "main")
	git(t, "pull", "-q", "--ff-only")
	if r := vcs(t, "git", "branch", "create", branch); r.code != 0 {
		t.Fatalf("branch create: %s", r.err)
	}
	writeFile(t, "live-"+stamp+".txt", "live")
	git(t, "add", ".")
	if r := vcs(t, "git", "commit", "-m", "Add a live file"); r.code != 0 {
		t.Fatalf("commit: %s", r.err)
	}
	if r := vcs(t, "git", "push", "-u", "origin", branch); r.code != 0 {
		t.Fatalf("push: %s", r.err)
	}
	t.Run("a bad branch name", func(t *testing.T) {
		if r := vcs(t, "git", "branch", "create", "Bad Name"); r.code == 0 {
			t.Error("a bad branch name went through")
		}
	})
	n := numberOf(t, ok(t, "pr", "create", "Live pull request", "--head", branch, "--body", "A live test change.", "--draft"))

	t.Run("draft and ready", func(t *testing.T) {
		contains(t, "list", ok(t, "pr", "list").out, "Live pull request")
		ok(t, "pr", "ready", n)
		ok(t, "pr", "draft", n)
		ok(t, "pr", "ready", n)
	})
	t.Run("edit", func(t *testing.T) {
		ok(t, "pr", "edit", n, "--title", "Live pull request renamed")
		contains(t, "list", ok(t, "pr", "list").out, "renamed")
	})
	t.Run("status and statuses", func(t *testing.T) {
		contains(t, "status", ok(t, "pr", "status", n).out, "open")
		ok(t, "pr", "statuses", n)
	})
	t.Run("bind", func(t *testing.T) {
		ok(t, "pr", "bind", n)
	})
	t.Run("tick", func(t *testing.T) {
		ok(t, "pr", "tick", "--dry-run")
	})
	t.Run("heal marks", func(t *testing.T) {
		ok(t, "pr", "heal", "ensure")
		ok(t, "pr", "heal", "add", n)
		ok(t, "pr", "heal", "remove", n)
	})
	t.Run("merge", func(t *testing.T) {
		ok(t, "pr", "merge", n, "--method", "squash", "--delete-branch")
		contains(t, "status", ok(t, "pr", "status", n).out, "merged")
	})
	t.Run("close", func(t *testing.T) {
		m := landChange(t, "fix/live-close-"+stamp, "close-"+stamp+".txt", "close", false)
		ok(t, "pr", "close", m)
		contains(t, "status", ok(t, "pr", "status", m).out, "closed")
		// A closed pull request can't change draft state. See issue 110.
		exits(t, 1, "pr", "ready", m)
		exits(t, 1, "pr", "draft", m)
	})
	t.Run("merge guard", func(t *testing.T) {
		vcs(t, "git", "branch", "switch", "main")
		if r := vcs(t, "git", "merge", "--ff-only", "origin/main"); r.code == 0 {
			t.Error("a merge on main went through")
		}
	})
	vcs(t, "git", "branch", "switch", "main")
	git(t, "pull", "-q", "--ff-only")
}

const workflow = `name: live
on:
  push:
  pull_request:
  workflow_dispatch:
jobs:
  hello:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello from the live test
`

// ensureWorkflow lands the workflow file on main the first time.
func ensureWorkflow(t *testing.T) {
	t.Helper()
	vcs(t, "git", "branch", "switch", "main")
	git(t, "pull", "-q", "--ff-only")
	if _, err := os.Stat(filepath.Join(cloneDir, ".forgejo/workflows/live.yml")); err == nil {
		return
	}
	landChange(t, "ci/live-workflow-"+stamp, ".forgejo/workflows/live.yml", workflow, true)
}

func TestCI(t *testing.T) {
	ensureWorkflow(t)
	ok(t, "workflow", "dispatch", "live.yml", "--ref", "main")
	var run string
	poll(t, 4*time.Minute, "a finished run of live.yml", func() bool {
		for _, r := range rows(ok(t, "ci", "runs", "--limit", "10").out) {
			if len(r) > 2 && r[2] == "success" {
				run = r[0]
				return true
			}
		}
		return false
	})
	t.Run("jobs", func(t *testing.T) {
		r := ok(t, "ci", "jobs", run)
		contains(t, "jobs", r.out, "hello")
	})
	var job string
	for _, r := range rows(ok(t, "ci", "jobs", run).out) {
		if len(r) > 2 && r[1] == "hello" {
			job = r[0]
		}
	}
	t.Run("log", func(t *testing.T) {
		if job == "" {
			t.Fatal("no job")
		}
		contains(t, "log", ok(t, "ci", "log", job, "--tail", "50").out, "Job succeeded")
	})
	t.Run("artifacts", func(t *testing.T) {
		ok(t, "ci", "artifacts", run)
	})
	t.Run("rerun makes a new attempt", func(t *testing.T) {
		before := ok(t, "ci", "jobs", run).out
		ok(t, "ci", "rerun", run)
		poll(t, 3*time.Minute, "attempt 2 of the job", func() bool {
			for _, r := range rows(ok(t, "ci", "jobs", run).out) {
				if len(r) > 3 && r[1] == "hello" && r[3] == "2" && r[2] == "success" {
					return true
				}
			}
			return false
		})
		_ = before
	})
	t.Run("rerun failed needs a failed job", func(t *testing.T) {
		exits(t, 1, "ci", "rerun", run, "--failed")
	})
	t.Run("failed and watch on a pull request", func(t *testing.T) {
		n := landChange(t, "fix/live-watch-"+stamp, "watch-"+stamp+".txt", "watch", false)
		r := vcs(t, target("pr", "watch", n)...)
		if r.code != 0 {
			t.Errorf("pr watch: exit %d\n%s\n%s", r.code, r.out, r.err)
		}
		ok(t, "ci", "failed", n)
		ok(t, "pr", "merge", n, "--method", "squash", "--delete-branch")
		vcs(t, "git", "branch", "switch", "main")
		git(t, "pull", "-q", "--ff-only")
	})
}

func TestSecretsAndPackages(t *testing.T) {
	name := "LIVE_" + stamp
	t.Run("secret from a file", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "s")
		os.WriteFile(f, []byte("made-up-value"), 0o600)
		ok(t, "secret", "set", name, "--value-file", f)
		contains(t, "list", ok(t, "secret", "list").out, name)
		ok(t, "secret", "delete", name)
		if strings.Contains(ok(t, "secret", "list").out, name) {
			t.Error("the secret is still there")
		}
	})
	t.Run("package", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "p.bin")
		os.WriteFile(f, []byte("payload"), 0o644)
		pkg := "live-" + stamp
		defer vcs(t, target("package", "delete", pkg, "0.0.2", "--allow-newest")...)
		defer vcs(t, target("package", "delete", pkg, "0.0.1", "--allow-newest")...)
		contains(t, "dry run", ok(t, "package", "upload", pkg, "0.0.1", f, "--dry-run").out, "dry-run")
		ok(t, "package", "upload", pkg, "0.0.1", f)
		contains(t, "list", ok(t, "package", "list").out, pkg)
		ok(t, "package", "upload", pkg, "0.0.2", f)
		ok(t, "package", "delete", pkg, "0.0.1")
		ok(t, "package", "delete", pkg, "0.0.2", "--allow-newest")
	})
}

// These verbs only read. A token without the scope gets a clear refusal, not a crash.
func TestReadOnlyVerbs(t *testing.T) {
	for _, args := range [][]string{
		{"protection", "list"},
		{"board", "list"},
		{"stopwatch", "list"},
		{"stopwatch", "list", "--check"},
		{"session", "status"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			r := vcs(t, target(args...)...)
			if r.code != 0 && r.code != 77 {
				t.Errorf("exit %d\n%s", r.code, r.err)
			}
		})
	}
	t.Run("runners list", func(t *testing.T) {
		r := vcs(t, target("runners", "list")...)
		if r.code != 0 && r.code != 77 && r.code != 1 {
			t.Errorf("exit %d\n%s", r.code, r.err)
		}
	})
}
