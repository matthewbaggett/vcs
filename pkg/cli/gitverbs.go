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
	if len(args) > 0 && args[0] == "-C" {
		if len(args) < 2 || args[1] == "" {
			return usage(env, v)
		}
		g.dir, args = args[1], args[2:]
	}
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
	dir string // the -C directory. Empty means the current one
}

// git puts -C <dir> in front of the arguments of a git call, when the verb got one.
func (g gitRunner) git(args []string) []string {
	if g.dir == "" {
		return args
	}
	return append([]string{"-C", g.dir}, args...)
}

// query runs git and returns the trimmed standard output and the exit code.
func (g gitRunner) query(args ...string) (string, int) {
	var out bytes.Buffer
	code := g.env.Exec(g.ctx, "git", g.git(args), &out, &bytes.Buffer{})
	return strings.TrimSpace(out.String()), code
}

// run runs git and shows its whole result.
func (g gitRunner) run(args ...string) int {
	code := g.env.Exec(g.ctx, "git", g.git(args), g.env.Stdout, g.env.Stderr)
	if code != 0 && len(args) > 0 {
		fmt.Fprintf(g.env.Stderr, "vcs: git %s failed with exit %d. Read the output above before the next step.\n", args[0], code)
	}
	return code
}

func (g gitRunner) branchName() string {
	b, code := g.query("symbolic-ref", "--quiet", "--short", "HEAD")
	if code != 0 {
		return ""
	}
	return b
}

// remote reads the origin address without the network. It returns the host name and the owner/name path.
func (g gitRunner) remote() (host, repo string, ok bool) {
	url, code := g.query("remote", "get-url", "origin")
	if code != 0 {
		return "", "", false
	}
	h, r, err := forge.ParseRemote(url)
	return h, r, err == nil
}

// forgeKind names the forge as the scripts do: github for github.com, forgejo for any other host.
func forgeKind(host string) string {
	switch strings.ToLower(host) {
	case "github.com", "www.github.com":
		return "github"
	}
	return "forgejo"
}

// note prints which forge origin is, on stderr, before the git call. The scripts print it too.
func (g gitRunner) note() {
	if h, _, ok := g.remote(); ok {
		fmt.Fprintf(g.env.Stderr, "vcs: host=%s\n", forgeKind(h))
	}
}

// defaultBranch finds the default branch of origin without the network. It reads origin/HEAD, tries
// main and master, then falls back to main.
func (g gitRunner) defaultBranch() string {
	if ref, code := g.query("symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); code == 0 && ref != "" {
		return strings.TrimPrefix(ref, "origin/")
	}
	for _, b := range []string{"main", "master"} {
		for _, ref := range []string{"refs/remotes/origin/" + b, "refs/heads/" + b} {
			if _, code := g.query("show-ref", "--verify", "--quiet", ref); code == 0 {
				return b
			}
		}
	}
	return "main"
}

// prHint says how to open the pull request for a branch that a push just sent.
func (g gitRunner) prHint(branch string) {
	h, repo, ok := g.remote()
	if !ok {
		return
	}
	base := g.defaultBranch()
	if branch == base {
		return
	}
	if forgeKind(h) == "github" {
		fmt.Fprintf(g.env.Stdout, "vcs: host=github. Open the pull request: vcs pr create --head %s, or open https://%s/%s/compare/%s...%s?expand=1\n", branch, h, repo, base, branch)
		return
	}
	fmt.Fprintf(g.env.Stdout, "vcs: host=forgejo. Open the pull request: vcs pr create <title> --body-file <file> --draft, or open https://%s/%s/compare/%s...%s\n", h, repo, base, branch)
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
			return g.refuse(ExitUsage, "--no-verify skips the git hooks, which run the same steps as CI. Drop the flag and run the command again. If a hook fails, read what it reports and fix that. To run one hook alone: git hook run pre-commit")
		}
		if shortN && strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
			// The letters before m, F, C, c, or S can hold -n. A later letter starts a value.
			pre := a[1:]
			if i := strings.IndexAny(pre, "mFCcS"); i >= 0 {
				pre = pre[:i]
			}
			if strings.Contains(pre, "n") {
				return g.refuse(ExitUsage, "-n skips the git hooks, as --no-verify does. Drop the flag and run the command again. If a hook fails, read what it reports and fix that. To run one hook alone: git hook run pre-commit")
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
	return g.lintBranch(b)
}

// lintBranch runs the project's scripts/lint-branch-name.sh on a branch name, when the project has it.
func (g gitRunner) lintBranch(b string) int {
	top, code := g.query("rev-parse", "--show-toplevel")
	if s := filepath.Join(top, "scripts", "lint-branch-name.sh"); code == 0 && isExec(s) {
		var out bytes.Buffer
		if g.env.Exec(g.ctx, s, []string{b}, &out, &out) != 0 {
			fmt.Fprint(g.env.Stderr, out.String())
			return g.refuse(ExitFail, "check branch-name failed in scripts/lint-branch-name.sh. Rename the branch")
		}
		g.ok("branch-name")
		return -1
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
func (g gitRunner) text(msg, what string) int {
	if r := guard.Credit(msg); len(r) > 0 {
		return g.refuse(ExitLint, "guard: %s. Delete that line from the %s, then run the command again", r[0], what)
	}
	g.ok("attribution")
	if code := lintText(g.env, msg); code != ExitOK {
		fmt.Fprintf(g.env.Stderr, "vcs: Rewrite the %s with the fixes above, then run the command again.\n", what)
		return code
	}
	g.ok("prose")
	return -1
}

// wiringScript finds the wiring check the scripts use. A project holds it in its agent-standards
// submodule. The agent-standards checkout holds it in hooks/.
func wiringScript(top string) string {
	if s := filepath.Join(top, ".claude", "agent-standards", "hooks", "check-wiring.sh"); isFile(s) {
		return s
	}
	if _, err := os.Lstat(filepath.Join(top, ".claude", "agent-standards")); err != nil &&
		isFile(filepath.Join(top, "bin", "install.sh")) {
		if s := filepath.Join(top, "hooks", "check-wiring.sh"); isFile(s) {
			return s
		}
	}
	return ""
}

// wiring runs the wiring check, as vcs-lib.sh does. It returns 0 when it passes or the check is off
// or absent, and 1 when it fails. The output of a failure goes to stderr. AGENT_STANDARDS_WIRING_GUARD=off
// skips the check on purpose.
func (g gitRunner) wiring(top string, quiet bool) int {
	skip := func(why string) int {
		if !quiet {
			fmt.Fprintf(g.env.Stdout, "vcs: check wiring: skipped (%s)\n", why)
		}
		return 0
	}
	if g.env.Getenv != nil && g.env.Getenv("AGENT_STANDARDS_WIRING_GUARD") == "off" {
		return skip("AGENT_STANDARDS_WIRING_GUARD=off")
	}
	s := wiringScript(top)
	if s == "" {
		return skip("no check-wiring.sh")
	}
	var out bytes.Buffer
	if g.env.Exec(g.ctx, "env", []string{"CLAUDE_PROJECT_DIR=" + top, "bash", s, "--ci"}, &out, &out) != 0 {
		fmt.Fprint(g.env.Stderr, out.String())
		return 1
	}
	return 0
}

// wiringNotes says how to repair the wiring and how to run the check alone. It prints them when a
// wiring check fails and stops a verb.
func (g gitRunner) wiringNotes(top string) {
	s := wiringScript(top)
	if s == "" {
		return
	}
	rel := func(p string) string {
		if r, err := filepath.Rel(top, p); err == nil {
			return r
		}
		return p
	}
	install := filepath.Join(filepath.Dir(filepath.Dir(s)), "bin", "install.sh")
	fmt.Fprintf(g.env.Stderr, "vcs: To repair the wiring, run from %s: %s\n", top, rel(install))
	fmt.Fprintf(g.env.Stderr, "vcs: Read what the installer changed, then commit .claude/ with any submodule bump. To run only this check: %s --ci\n", rel(s))
}

func (g gitRunner) wiringOff() bool {
	return g.env.Getenv != nil && g.env.Getenv("AGENT_STANDARDS_WIRING_GUARD") == "off"
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// project runs the project's own check scripts, when the project has them.
func (g gitRunner) project(verb string) int {
	top, code := g.query("rev-parse", "--show-toplevel")
	if code != 0 {
		return -1
	}
	if g.wiring(top, false) != 0 {
		g.wiringNotes(top)
		return g.refuse(ExitFail, "check wiring failed in %s. Fix each line above, or run the installer, then run the command again", top)
	}
	if wiringScript(top) != "" && !g.wiringOff() {
		g.ok("wiring")
	}
	return g.projectScript(top, code, verb)
}

// projectScript runs scripts/vcs-checks.sh <verb>, when the project has it.
func (g gitRunner) projectScript(top string, code int, verb string) int {
	if code != 0 {
		return -1
	}
	if s := filepath.Join(top, "scripts", "vcs-checks.sh"); isExec(s) {
		if g.env.Exec(g.ctx, s, []string{verb}, g.env.Stdout, g.env.Stderr) != 0 {
			fmt.Fprintf(g.env.Stderr, "vcs: To run only this check, from %s: ./scripts/vcs-checks.sh %s\n", top, verb)
			return g.refuse(ExitFail, "check project (%s) failed. Fix what the lines above report, run the check alone until it passes, then run the command again", verb)
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
		if c := g.text(msg, "commit message"); c >= 0 {
			return c
		}
	} else {
		fmt.Fprintln(g.env.Stdout, "vcs: check attribution: skipped (no message in the arguments)")
		fmt.Fprintln(g.env.Stdout, "vcs: check prose: skipped (no message in the arguments)")
	}
	if c := g.project("commit"); c >= 0 {
		return c
	}
	g.note()
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
	g.note()
	code := g.run(append([]string{"push"}, args...)...)
	if code == 0 {
		if b := g.branchName(); b != "" {
			g.prHint(b)
		}
	}
	return code
}

func (g gitRunner) merge(v Verb, args []string) int {
	if len(args) == 0 {
		return usage(g.env, v)
	}
	if c := g.bypass(v, args, false); c >= 0 {
		return c
	}
	control, cont := false, false
	for _, a := range args {
		switch a {
		case "--abort", "--quit":
			control = true
		case "--continue":
			control, cont = true, true
		}
	}
	top, topCode := g.query("rev-parse", "--show-toplevel")
	heal := false
	switch {
	case !control:
		if c := g.notDefault("merge"); c >= 0 {
			return c
		}
		if msg, found := message(args); found {
			if c := g.text(msg, "merge message"); c >= 0 {
				return c
			}
		}
		if topCode == 0 && g.wiring(top, false) != 0 {
			// The wiring can fail only because the branch lacks what main gained. The merge can be the
			// fix. So with a clean tree and no merge in progress, try the merge, then check again.
			dirty, _ := g.query("status", "--porcelain", "--untracked-files=no")
			_, inMerge := g.query("rev-parse", "--verify", "--quiet", "MERGE_HEAD")
			if dirty != "" || inMerge == 0 {
				return g.refuse(ExitFail, "check wiring failed, and the tree has changes or a merge in progress, so vcs can't try the merge. Commit or set the changes aside first")
			}
			fmt.Fprintln(g.env.Stderr, "vcs: check wiring failed before the merge. vcs runs the merge, then checks the wiring on the result")
			heal = true
		} else if topCode == 0 && wiringScript(top) != "" && !g.wiringOff() {
			g.ok("wiring")
		}
		if c := g.projectScript(top, topCode, "merge"); c >= 0 {
			return c
		}
	case cont:
		if topCode == 0 && g.wiring(top, true) != 0 {
			g.wiringNotes(top)
			return g.refuse(ExitFail, "check wiring failed in %s. Fix each line above, or run the installer, then run the command again", top)
		}
	}
	before, _ := g.query("rev-parse", "--verify", "--quiet", "HEAD")
	g.note()
	code := g.run(append([]string{"merge"}, args...)...)
	if heal && code == 0 {
		if g.wiring(top, true) != 0 {
			if before != "" {
				g.run("reset", "-q", "--hard", before)
			}
			return g.refuse(ExitFail, "check wiring still fails after the merge. vcs put the branch back. The merge doesn't fix the wiring")
		}
		g.ok("wiring (after the merge)")
	}
	return code
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
			// Read the local origin ref, as the script does. The check needs no network.
			if _, code := g.query("show-ref", "--verify", "--quiet", "refs/remotes/origin/"+b); code == 0 {
				return g.refuse(ExitFail, "origin already holds %s, and a rebase needs a force-push. Merge instead: vcs git merge origin/%s", b, g.defaultBranch())
			}
		}
		g.ok("not-published")
		if c := g.project("rebase"); c >= 0 {
			return c
		}
	}
	g.note()
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
		if c := g.lintBranch(names[0]); c >= 0 {
			return c
		}
		start := ""
		if len(names) == 2 {
			start = names[1]
		} else {
			d := g.defaultBranch()
			if _, code := g.query("show-ref", "--verify", "--quiet", "refs/remotes/origin/"+d); code == 0 {
				start = "origin/" + d
			} else {
				start = "HEAD"
			}
		}
		g.note()
		return g.run("switch", "--no-track", "-c", names[0], start)
	case "switch":
		if len(names) != 1 || force || remote || strings.HasPrefix(names[0], "-") {
			return usage(g.env, v)
		}
		g.note()
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
		old := g.branchName()
		if len(names) == 2 {
			old = names[0]
		}
		if old == "" {
			return g.refuse(ExitFail, "detached head: name the branch to rename")
		}
		d := g.defaultBranch()
		for _, n := range []string{old, next} {
			if n == d || n == "main" || n == "master" {
				return g.refuse(ExitFail, "refusing to rename to or from the default branch %s", n)
			}
		}
		if _, code := g.query("show-ref", "--verify", "--quiet", "refs/heads/"+old); code != 0 {
			return g.refuse(ExitFail, "no local branch %s", old)
		}
		if r := guard.Branch(next); len(r) > 0 {
			return g.refuse(ExitFail, "guard: %s", r[0])
		}
		if c := g.lintBranch(next); c >= 0 {
			return c
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
	held := g.originHolds(old)
	if !held || g.forgeOf() != "forgejo" {
		// No server rename: push the new name first, so the commits are never off the remote. Then
		// drop the old name.
		if c := g.run("branch", "-m", old, next); c != ExitOK {
			return c
		}
		if c := g.run("push", "-u", "origin", next); c != ExitOK {
			return c
		}
		if held {
			if c := g.run("push", "origin", "--delete", old); c != ExitOK {
				return c
			}
			fmt.Fprintf(g.env.Stderr, "vcs: origin no longer has %s. A pull request that used %s as its head is closed. Open one for %s with vcs pr create.\n", old, old, next)
		}
		return ExitOK
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
		fmt.Fprintf(g.env.Stderr, "vcs: the host refused to rename %s on origin. Nothing changed locally. %v\n", old, err)
		return ExitFail
	}
	fmt.Fprintf(g.env.Stderr, "vcs: origin renamed %s to %s. An open pull request keeps its number.\n", old, next)
	if c := g.run("branch", "-m", old, next); c != ExitOK {
		return c
	}
	if c := g.run("fetch", "origin"); c != ExitOK {
		fmt.Fprintf(g.env.Stderr, "vcs: origin renamed. Run git fetch origin, then git branch --set-upstream-to=origin/%s %s\n", next, next)
		return ExitOK
	}
	return g.run("branch", "--set-upstream-to=origin/"+next, next)
}

// originHolds says whether origin has a branch. It reads the local ref, then asks the server.
func (g gitRunner) originHolds(b string) bool {
	if _, code := g.query("show-ref", "--verify", "--quiet", "refs/remotes/origin/"+b); code == 0 {
		return true
	}
	_, code := g.query("ls-remote", "--exit-code", "--heads", "origin", b)
	return code == 0
}

// forgeOf names the forge of origin. With no origin it reads the home remote of the build.
func (g gitRunner) forgeOf() string {
	if h, _, ok := g.remote(); ok {
		return forgeKind(h)
	}
	if g.env.RemoteURL != nil {
		if url, err := g.env.RemoteURL(); err == nil {
			if h, _, err := forge.ParseRemote(url); err == nil {
				return forgeKind(h)
			}
		}
	}
	return "forgejo"
}
