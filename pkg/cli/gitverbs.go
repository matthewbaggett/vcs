package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/guard"
)

// gitVerb runs one of the checked git verbs. It takes the raw arguments, because git has flags
// that the other verbs reject. Each verb runs its checks, stops at the first failure, then runs git
// and returns the exit code of git.
func gitVerb(ctx context.Context, v Verb, args []string, env Env) int {
	if env.Exec == nil {
		fmt.Fprintln(env.Stderr, "vcs: this build can't run git")
		return ExitUnavailable
	}
	g := gitRunner{ctx: ctx, env: env}
	switch v.Name {
	case "commit":
		return g.commit(v, args)
	case "push":
		return g.push(v, args)
	case "merge":
		return g.merge(v, args)
	case "rebase":
		return g.rebase(v, args)
	case "branch":
		return g.branch(v, args)
	}
	return usage(env, v)
}

type gitRunner struct {
	ctx context.Context
	env Env
}

// query runs git and returns the trimmed standard output and the exit code.
func (g gitRunner) query(args ...string) (string, int) {
	var out bytes.Buffer
	code := g.env.Exec(g.ctx, "git", args, &out, &bytes.Buffer{})
	return strings.TrimSpace(out.String()), code
}

// run runs git and shows its whole result.
func (g gitRunner) run(args ...string) int {
	return g.env.Exec(g.ctx, "git", args, g.env.Stdout, g.env.Stderr)
}

func (g gitRunner) branchName() string {
	b, code := g.query("symbolic-ref", "--quiet", "--short", "HEAD")
	if code != 0 {
		return ""
	}
	return b
}

func (g gitRunner) ok(check string) { fmt.Fprintf(g.env.Stdout, "vcs: check %s: ok\n", check) }

func (g gitRunner) refuse(code int, format string, a ...any) int {
	fmt.Fprintf(g.env.Stderr, "vcs: "+format+"\n", a...)
	return code
}

// bypass refuses a flag that skips the git hooks.
func (g gitRunner) bypass(v Verb, args []string, shortN bool) int {
	for _, a := range args {
		if a == "--no-verify" {
			return g.refuse(ExitUsage, "--no-verify skips the git hooks, which run the same steps as CI. Fix what the hook reports")
		}
		if shortN && strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
			// The letters before m, F, C, c, or S can hold -n. A later letter starts a value.
			pre := a[1:]
			if i := strings.IndexAny(pre, "mFCcS"); i >= 0 {
				pre = pre[:i]
			}
			if strings.Contains(pre, "n") {
				return g.refuse(ExitUsage, "-n skips the git hooks, as --no-verify does. Fix what the hook reports")
			}
		}
	}
	g.ok("no-hook-bypass")
	return -1
}

// notDefault refuses to write on the default branch.
func (g gitRunner) notDefault(what string) int {
	if b := g.branchName(); b == "main" || b == "master" {
		return g.refuse(ExitFail, "this %s would land on %s. Work on a <type>/<short-description> branch: vcs git branch create feature/<desc>", what, b)
	}
	g.ok("not-default-branch")
	return -1
}

// branchOK checks the branch name against the type/short-description rule.
func (g gitRunner) branchOK() int {
	b := g.branchName()
	if b == "" {
		fmt.Fprintln(g.env.Stdout, "vcs: check branch-name: skipped (detached head)")
		return -1
	}
	if r := guard.Branch(b); len(r) > 0 {
		return g.refuse(ExitFail, "guard: %s. Rename it: vcs git branch rename <type>/<short-description>", r[0])
	}
	g.ok("branch-name")
	return -1
}

// message finds the text of a commit message in the git arguments.
func message(args []string) (string, bool) {
	var parts []string
	found := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-m" || a == "--message":
			if i+1 < len(args) {
				parts = append(parts, args[i+1])
				found = true
				i++
			}
		case strings.HasPrefix(a, "--message="):
			parts = append(parts, strings.TrimPrefix(a, "--message="))
			found = true
		case strings.HasPrefix(a, "-m") && !strings.HasPrefix(a, "--") && len(a) > 2:
			parts = append(parts, a[2:])
			found = true
		case a == "-F" || a == "--file":
			if i+1 < len(args) {
				if b, err := os.ReadFile(args[i+1]); err == nil {
					parts = append(parts, string(b))
					found = true
				}
				i++
			}
		case strings.HasPrefix(a, "--file="):
			if b, err := os.ReadFile(strings.TrimPrefix(a, "--file=")); err == nil {
				parts = append(parts, string(b))
				found = true
			}
		}
	}
	return strings.Join(parts, "\n\n"), found
}

// text checks a message for an AI credit line, then runs the prose lint.
func (g gitRunner) text(msg string) int {
	if r := guard.Credit(msg); len(r) > 0 {
		return g.refuse(ExitLint, "guard: %s", r[0])
	}
	g.ok("attribution")
	if code := lintText(g.env, msg); code != ExitOK {
		return code
	}
	g.ok("prose")
	return -1
}

// project runs the project's own check scripts, when the project has them.
func (g gitRunner) project(verb string) int {
	top, code := g.query("rev-parse", "--show-toplevel")
	if code != 0 {
		return -1
	}
	if s := filepath.Join(top, ".claude", "check-wiring.sh"); isExec(s) {
		if g.env.Exec(g.ctx, s, []string{"--ci"}, g.env.Stdout, g.env.Stderr) != 0 {
			return g.refuse(ExitFail, "check wiring failed. Fix the wiring, then run the command again")
		}
		g.ok("wiring")
	} else {
		fmt.Fprintln(g.env.Stdout, "vcs: check wiring: skipped (no .claude/check-wiring.sh)")
	}
	if s := filepath.Join(top, "scripts", "vcs-checks.sh"); isExec(s) {
		if g.env.Exec(g.ctx, s, []string{verb}, g.env.Stdout, g.env.Stderr) != 0 {
			return g.refuse(ExitFail, "check project failed. Fix what scripts/vcs-checks.sh printed")
		}
		g.ok("project")
	} else {
		fmt.Fprintln(g.env.Stdout, "vcs: check project: skipped (no scripts/vcs-checks.sh)")
	}
	return -1
}

func isExec(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

func (g gitRunner) commit(v Verb, args []string) int {
	if c := g.bypass(v, args, true); c >= 0 {
		return c
	}
	if c := g.notDefault("commit"); c >= 0 {
		return c
	}
	if c := g.branchOK(); c >= 0 {
		return c
	}
	if msg, found := message(args); found {
		if c := g.text(msg); c >= 0 {
			return c
		}
	} else {
		fmt.Fprintln(g.env.Stdout, "vcs: check attribution: skipped (no message in the arguments)")
		fmt.Fprintln(g.env.Stdout, "vcs: check prose: skipped (no message in the arguments)")
	}
	if c := g.project("commit"); c >= 0 {
		return c
	}
	return g.run(append([]string{"commit"}, args...)...)
}

func (g gitRunner) push(v Verb, args []string) int {
	for _, a := range args {
		short := strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a[1:], "f")
		if short || a == "--force-if-includes" {
			return g.refuse(ExitUsage, "force-push is off the table. Update the pull request with new commits on top. To fix a conflict, merge origin/main and push")
		}
	}
	if r := guard.Push(args); len(r) > 0 {
		return g.refuse(ExitUsage, "guard: %s. Update the pull request with new commits on top", r[0])
	}
	g.ok("no-force")
	if c := g.project("push"); c >= 0 {
		return c
	}
	if len(args) == 0 {
		if b := g.branchName(); b != "" {
			up, code := g.query("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
			// A branch cut from origin/main tracks origin/main, and a plain push then refuses.
			if code != 0 || up == "" || up[strings.Index(up+"/", "/")+1:] != b {
				args = []string{"-u", "origin", b}
			}
		}
	}
	return g.run(append([]string{"push"}, args...)...)
}

func (g gitRunner) merge(v Verb, args []string) int {
	if len(args) == 0 {
		return usage(g.env, v)
	}
	if c := g.bypass(v, args, false); c >= 0 {
		return c
	}
	control := false
	for _, a := range args {
		if a == "--abort" || a == "--continue" || a == "--quit" {
			control = true
		}
	}
	if !control {
		if c := g.notDefault("merge"); c >= 0 {
			return c
		}
		if msg, found := message(args); found {
			if c := g.text(msg); c >= 0 {
				return c
			}
		}
	}
	if c := g.project("merge"); c >= 0 && !control {
		return c
	}
	return g.run(append([]string{"merge"}, args...)...)
}

func (g gitRunner) rebase(v Verb, args []string) int {
	if len(args) == 0 {
		return usage(g.env, v)
	}
	if c := g.bypass(v, args, false); c >= 0 {
		return c
	}
	control := false
	for _, a := range args {
		switch a {
		case "--abort", "--continue", "--skip", "--quit":
			control = true
		case "-i", "--interactive":
			return g.refuse(ExitUsage, "nobody answers an editor, so a rebase cannot be interactive")
		}
	}
	if !control {
		if c := g.notDefault("rebase"); c >= 0 {
			return c
		}
		if b := g.branchName(); b != "" {
			if _, code := g.query("ls-remote", "--exit-code", "--heads", "origin", b); code == 0 {
				return g.refuse(ExitFail, "origin holds %s, and a rebase needs a force-push. Merge instead: vcs git merge origin/main", b)
			}
		}
		g.ok("not-published")
		if c := g.project("rebase"); c >= 0 {
			return c
		}
	}
	return g.run(append([]string{"rebase"}, args...)...)
}

func (g gitRunner) branch(v Verb, args []string) int {
	if len(args) == 0 {
		return usage(g.env, v)
	}
	act, rest := args[0], args[1:]
	var names []string
	var remote, force bool
	for _, a := range rest {
		switch a {
		case "--remote":
			remote = true
		case "--force":
			force = true
		default:
			names = append(names, a)
		}
	}
	switch act {
	case "create":
		if len(names) < 1 || len(names) > 2 || force || remote || strings.HasPrefix(names[0], "-") {
			return usage(g.env, v)
		}
		if r := guard.Branch(names[0]); len(r) > 0 {
			return g.refuse(ExitFail, "guard: %s", r[0])
		}
		g.ok("branch-name")
		start := ""
		if len(names) == 2 {
			start = names[1]
		} else if _, code := g.query("rev-parse", "--verify", "--quiet", "refs/remotes/origin/main"); code == 0 {
			start = "origin/main"
		}
		a := []string{"switch", "-c", names[0]}
		if start != "" {
			a = append(a, start)
		}
		return g.run(a...)
	case "switch":
		if len(names) != 1 || force || remote || strings.HasPrefix(names[0], "-") {
			return usage(g.env, v)
		}
		return g.run("switch", names[0])
	case "list":
		if len(names) != 0 || force || remote {
			return usage(g.env, v)
		}
		return g.run("branch", "-vv")
	case "rename":
		if force || len(names) < 1 || len(names) > 2 || strings.HasPrefix(names[len(names)-1], "-") {
			return usage(g.env, v)
		}
		next := names[len(names)-1]
		if r := guard.Branch(next); len(r) > 0 {
			return g.refuse(ExitFail, "guard: %s", r[0])
		}
		g.ok("branch-name")
		old := g.branchName()
		if len(names) == 2 {
			old = names[0]
		}
		if old == "main" || old == "master" || old == "" {
			return g.refuse(ExitFail, "never rename the default branch, or a detached head")
		}
		if remote {
			return g.renameRemote(old, next)
		}
		return g.run("branch", "-m", old, next)
	case "delete":
		if len(names) != 1 || strings.HasPrefix(names[0], "-") {
			return usage(g.env, v)
		}
		n := names[0]
		if n == "main" || n == "master" || n == g.branchName() {
			return g.refuse(ExitFail, "never delete the default branch or the branch you are on")
		}
		flag := "-d"
		if force {
			flag = "-D"
		}
		if c := g.run("branch", flag, n); c != 0 {
			return c
		}
		if remote {
			return g.run("push", "origin", "--delete", n)
		}
		return ExitOK
	}
	return usage(g.env, v)
}

// renameRemote renames a branch on the server, then locally. The server call moves the head of
// each open pull request, so the pull request keeps its number. A branch that origin doesn't hold
// gets no server call: the new name is pushed instead. The driver check runs before any change.
func (g gitRunner) renameRemote(old, next string) int {
	if _, held := g.query("ls-remote", "--exit-code", "--heads", "origin", old); held != 0 {
		if c := g.run("branch", "-m", old, next); c != ExitOK {
			return c
		}
		return g.run("push", "-u", "origin", next)
	}
	if g.env.RemoteURL == nil || g.env.NewClient == nil {
		fmt.Fprintln(g.env.Stderr, "vcs: this build can't reach a forge")
		return ExitUnavailable
	}
	url, err := g.env.RemoteURL()
	if err != nil {
		fmt.Fprintf(g.env.Stderr, "vcs: no home remote. %v\n", err)
		return ExitFail
	}
	host, repo, err := forge.ParseRemote(url)
	if err != nil {
		fmt.Fprintf(g.env.Stderr, "vcs: %v\n", err)
		return ExitFail
	}
	c, err := g.env.NewClient(host, repo)
	if err != nil {
		fmt.Fprintf(g.env.Stderr, "vcs: %s: %v\n", host, err)
		if errors.Is(err, forge.ErrUnsupported) {
			return ExitUnavailable
		}
		return ExitFail
	}
	r, ok := c.(forge.BranchRenamer)
	if !ok {
		fmt.Fprintf(g.env.Stderr, "vcs: the driver for %s can't rename a branch on the server\n", host)
		return ExitUnavailable
	}
	if err := r.BranchRename(g.ctx, old, next); err != nil {
		fmt.Fprintf(g.env.Stderr, "vcs: the host refused to rename %s. Nothing changed locally. %v\n", old, err)
		return ExitFail
	}
	if c := g.run("branch", "-m", old, next); c != ExitOK {
		return c
	}
	if c := g.run("fetch", "origin"); c != ExitOK {
		fmt.Fprintf(g.env.Stderr, "vcs: origin renamed. Run git fetch origin, then git branch --set-upstream-to=origin/%s %s\n", next, next)
		return ExitOK
	}
	return g.run("branch", "--set-upstream-to=origin/"+next, next)
}
