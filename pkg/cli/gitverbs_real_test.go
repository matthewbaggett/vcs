package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/lint"
)

// This file ports each case of tools/forgejo/vcs-git.test.sh of agent-standards to Go, with real
// git and no simulator. Each case runs the verb in a scratch repository. The address of origin
// says github.com or git.grey.ooo, and a separate push address points at a local bare
// repository. So one test covers the GitHub host and the Forgejo host with no network.
//
// The shell test runs 145 checks. This test runs the same 145, in the same order, and each
// check is a subtest. A check that differs on purpose carries "[deviation: ...]" in its name:
//
//   - The binary says "vcs pr create" and "vcs git merge", where the scripts said "vcs-pr.sh"
//     and "vcs-merge.sh".
//   - The guard of the binary checks a branch name before the project's lint script runs. So a
//     name that breaks the type list reports "guard:", and a name that only breaks the lint script
//     reports "check branch-name failed". The extra checks at the end cover that second case.
//   - A prose finding reports "finding(s)", not "breaks the prose lint".
//   - The wiring text says "can't try the merge", not "cannot try the merge".
//   - The Forgejo branch rename goes through the driver, not through `tea`. The check reads the call
//     that the fake driver got.
//   - A branch name starts with fix/ where the shell test used feat/. The guard of the binary has
//     no feat type.
//
// The checks after the 145 are extra. They cover a repo with no origin and the lint script path.

// utilizeChecker flags the word Utilize, as the stub lint-prose.sh does.
type utilizeChecker struct{}

func (utilizeChecker) Name() string { return "utilize" }

func (utilizeChecker) Check(text string) ([]lint.Finding, error) {
	if strings.Contains(text, "Utilize") {
		return []lint.Finding{{Checker: "utilize", Line: 1, Column: 1, Rule: "stub", Message: "Utilize"}}, nil
	}
	return nil, nil
}

// bareRenamer is a fake Forgejo driver. It renames the branch in the bare repo of the test repo.
type bareRenamer struct {
	fake
	bare string
	log  *[]string
}

func (b *bareRenamer) BranchRename(_ context.Context, old, name string) error {
	*b.log = append(*b.log, old+" "+name)
	if out, err := exec.Command("git", "-C", b.bare, "branch", "-m", old, name).CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

// rig is the scratch space of the port.
type rig struct {
	t       *testing.T
	tmp     string
	n, nx   int      // checks run: the ported ones, and the extra ones
	extra   bool     // the checks that run now are extra
	rc      int      // the exit code of the last run
	out     string   // the whole output of the last run
	renames []string // each call that the fake driver got
}

func newRig(t *testing.T) *rig {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX", "GIT_COMMON_DIR"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	for k, v := range map[string]string{
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.com",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.com",
		"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null",
		// No editor opens for a merge commit, and no fetch reaches the network.
		"GIT_EDITOR": "true", "GIT_MERGE_AUTOEDIT": "no", "GIT_TERMINAL_PROMPT": "0",
		"GIT_ALLOW_PROTOCOL": "file", "AGENT_STANDARDS_WIRING_GUARD": "",
	} {
		t.Setenv(k, v)
	}
	return &rig{t: t, tmp: t.TempDir()}
}

// git runs real git in a directory. A failure ends the test.
func (r *rig) git(dir string, args ...string) string {
	r.t.Helper()
	out, err := r.gitErr(dir, args...)
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func (r *rig) gitErr(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r *rig) write(path, text string, mode os.FileMode) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), mode); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) appendTo(path, text string) {
	r.t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		r.t.Fatal(err)
	}
	defer f.Close()
	f.WriteString(text)
}

// commitRaw makes a commit with no hooks, for the set-up of a case.
func (r *rig) commitRaw(dir, msg string) {
	r.git(dir, "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", msg)
}

// newRepo makes a repository on main with one commit, a bare push target, a conforming branch
// lint, and a wiring stub. It returns the path.
func (r *rig) newRepo(name, origin string) string {
	r.t.Helper()
	d, bare := filepath.Join(r.tmp, name), filepath.Join(r.tmp, name+".bare")
	r.git(r.tmp, "init", "-q", "-b", "main", "--bare", bare)
	r.git(r.tmp, "init", "-q", "-b", "main", d)
	r.git(d, "remote", "add", "origin", origin)
	r.git(d, "config", "remote.origin.pushurl", bare)
	r.write(filepath.Join(d, "scripts", "lint-branch-name.sh"), `#!/usr/bin/env bash
printf '%s' "$1" | grep -Eq '^(feat|fix|docs)/[a-z0-9]+(-[a-z0-9]+)*$' && exit 0
echo "branch-name: '$1' is not <type>/<short-description>."
exit 1
`, 0o755)
	// The stub fails on a WIRING_BAD file. It also fails for each path that wiring.need names and the tree lacks.
	r.write(filepath.Join(d, "hooks", "check-wiring.sh"), `#!/usr/bin/env bash
[ -f "$CLAUDE_PROJECT_DIR/WIRING_BAD" ] && { echo "wiring is broken"; exit 1; }
if [ -f "$CLAUDE_PROJECT_DIR/wiring.need" ]; then
  while read -r p; do
    [ -e "$CLAUDE_PROJECT_DIR/$p" ] || { echo "  - $p is missing"; bad=1; }
  done <"$CLAUDE_PROJECT_DIR/wiring.need"
  [ -z "${bad:-}" ] || exit 1
fi
exit 0
`, 0o644)
	r.write(filepath.Join(d, "bin", "install.sh"), "", 0o644)
	r.write(filepath.Join(d, ".vale.ini"), "", 0o644)
	r.write(filepath.Join(d, "file.txt"), "one\n", 0o644)
	r.git(d, "add", "-A")
	r.commitRaw(d, "Start the repo")
	r.git(d, "push", "-q", "origin", "main")
	r.git(d, "update-ref", "refs/remotes/origin/main", "main")
	return d
}

// run runs `vcs git <args>` with dir as the current directory. It sets rc and out.
func (r *rig) run(dir string, args ...string) {
	r.t.Helper()
	var buf bytes.Buffer
	env := Env{
		Stdout: &buf, Stderr: &buf, Getenv: os.Getenv,
		Checkers: []lint.Checker{utilizeChecker{}},
		Exec: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Dir, cmd.Stdout, cmd.Stderr = dir, stdout, stderr
			if err := cmd.Run(); err != nil {
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					return ee.ExitCode()
				}
				fmt.Fprintf(stderr, "vcs: %v\n", err)
				return 1
			}
			return 0
		},
		RemoteURL: func() (string, error) {
			out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
			return strings.TrimSpace(string(out)), err
		},
		NewClient: func(string, string) (forge.Client, error) {
			return &bareRenamer{bare: dir + ".bare", log: &r.renames}, nil
		},
	}
	r.rc = Run(context.Background(), append([]string{"git"}, args...), env)
	r.out = buf.String()
}

// count tells which counter a check adds to.
func (r *rig) count() {
	if r.extra {
		r.nx++
		return
	}
	r.n++
}

// check is one case. It is a subtest, so a failure names the case.
func (r *rig) check(name string, ok bool, detail string) {
	r.t.Helper()
	r.count()
	r.t.Run(name, func(t *testing.T) {
		if !ok {
			t.Errorf("%s", detail)
		}
	})
}

func (r *rig) wantRC(name string, want int) {
	r.t.Helper()
	r.check(name+": exit", r.rc == want, fmt.Sprintf("exit %d, want %d\n%s", r.rc, want, r.out))
}

func (r *rig) wantOut(name, text string) {
	r.t.Helper()
	r.check(name+": output has "+text, strings.Contains(r.out, text), fmt.Sprintf("output lacks %q\n%s", text, r.out))
}

func (r *rig) noOut(name, text string) {
	r.t.Helper()
	r.check(name+": output has no "+text, !strings.Contains(r.out, text), fmt.Sprintf("output holds %q\n%s", text, r.out))
}

func (r *rig) head(dir string) string { return r.git(dir, "rev-parse", "HEAD") }

func (r *rig) hasRef(dir, ref string) bool {
	_, err := r.gitErr(dir, "rev-parse", "--verify", "-q", ref)
	return err == nil
}

func (r *rig) current(dir string) string { return r.git(dir, "symbolic-ref", "--short", "HEAD") }

func TestGitVerbsMatchScripts(t *testing.T) {
	r := newRig(t)
	GH := r.newRepo("gh", "https://github.com/acme/widgets.git")
	FG := r.newRepo("fg", "https://git.grey.ooo/Grey.ooo/widgets.git")
	ghBare, fgBare := GH+".bare", FG+".bare"

	// --- the host ------------------------------------------------------------
	hostOf := func(url string) string {
		d := filepath.Join(r.tmp, "h")
		os.RemoveAll(d)
		r.git(r.tmp, "init", "-q", "-b", "main", d)
		r.git(d, "remote", "add", "origin", url)
		g := gitRunner{ctx: context.Background(), dir: d, env: Env{Exec: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout, cmd.Stderr = stdout, stderr
			if cmd.Run() != nil {
				return 1
			}
			return 0
		}}}
		h, _, ok := g.remote()
		if !ok {
			return "no origin"
		}
		return forgeKind(h)
	}
	for _, u := range []string{"https://github.com/a/b.git", "git@github.com:a/b.git", "ssh://git@github.com/a/b.git", "https://user@github.com/a/b", "https://GitHub.com/a/b"} {
		got := hostOf(u)
		r.check("host: "+u+" is github", got == "github", "got "+got)
	}
	for _, u := range []string{"https://git.grey.ooo/a/b.git", "git@git.grey.ooo:a/b.git", "ssh://git@git.example.org:2222/a/b.git", "https://github.com.example.org/a/b.git", "https://example.com/github.com/b.git"} {
		got := hostOf(u)
		r.check("host: "+u+" is forgejo", got == "forgejo", "got "+got)
	}
	remoteParts := func(d string) string {
		g := gitRunner{ctx: context.Background(), dir: d, env: Env{Exec: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout, cmd.Stderr = stdout, stderr
			if cmd.Run() != nil {
				return 1
			}
			return 0
		}}}
		h, repo, _ := g.remote()
		return h + " " + repo
	}
	// The binary joins owner and name as one path, so "acme/widgets" stands where the script printed "acme widgets".
	got := remoteParts(GH)
	r.check("remote parts of the github repo [deviation: owner/name is one path]", got == "github.com acme/widgets", got)
	got = remoteParts(FG)
	r.check("remote parts of the forgejo repo [deviation: owner/name is one path]", got == "git.grey.ooo Grey.ooo/widgets", got)

	// --- vcs git branch ------------------------------------------------------
	for _, c := range []struct{ name, d string }{{"GH", GH}, {"FG", FG}} {
		r.run(c.d, "branch", "create", "fix/first-change")
		r.wantRC(c.name+" branch create", 0)
		cur := r.current(c.d)
		r.check(c.name+" branch create: on the new branch", cur == "fix/first-change", cur)
	}
	r.run(GH, "branch", "create", "Bad_Name")
	r.wantRC("branch create refuses a bad name", 1)
	r.wantOut("branch create names the check [deviation: the guard reports first, as guard:]", "guard:")
	r.run(GH, "branch", "create", "-x")
	r.wantRC("branch create refuses a dash name", 64)
	r.run(GH, "branch", "delete", "main")
	r.wantRC("branch delete refuses main", 1)
	r.run(GH, "branch", "delete", "fix/first-change")
	r.wantRC("branch delete refuses the current branch", 1)
	r.run(GH, "branch", "bogus")
	r.wantRC("branch usage", 64)
	r.run(GH, "branch", "list")
	r.wantRC("branch list", 0)
	r.wantOut("branch list", "fix/first-change")
	r.run(GH, "branch", "rename", "fix/first-change", "Bad_Name")
	r.wantRC("branch rename checks the new name", 1)
	r.run(GH, "branch", "rename", "fix/first-change", "fix/renamed-change")
	r.wantRC("branch rename", 0)
	r.run(GH, "branch", "rename", "fix/one-arg-rename")
	r.wantRC("branch rename with one argument renames the current branch", 0)
	r.run(GH, "branch", "rename", "fix/renamed-change")
	r.wantRC("branch rename back, one argument", 0)
	r.run(GH, "branch", "rename", "-bad")
	r.wantRC("branch rename refuses a name with a leading dash", 64)
	r.run(GH, "branch", "rename", "fix/renamed-change", "-x")
	r.wantRC("branch rename refuses a dash name", 64)
	r.run(GH, "branch", "rename", "main", "fix/from-main")
	r.wantRC("branch rename refuses the default branch as old", 1)
	r.run(GH, "branch", "rename", "fix/renamed-change", "main")
	r.wantRC("branch rename refuses the default branch as new", 1)
	r.run(GH, "branch", "rename", "fix/missing-branch", "fix/other-name")
	r.wantRC("branch rename refuses a missing branch", 1)
	r.wantOut("branch rename names the missing branch", "no local branch")
	// --remote: push the new name, delete the old name on origin.
	r.run(GH, "branch", "create", "fix/remote-old", "main")
	r.git(GH, "push", "-q", "origin", "fix/remote-old")
	r.run(GH, "branch", "rename", "--remote", "fix/remote-old", "Bad_Name")
	r.wantRC("branch rename --remote checks the new name", 1)
	r.check("rename --remote with a bad name: the old remote branch stays", r.hasRef(ghBare, "refs/heads/fix/remote-old"), "the old remote branch is gone")
	r.run(GH, "branch", "rename", "--remote", "fix/remote-old", "fix/remote-new")
	r.wantRC("branch rename --remote", 0)
	r.wantOut("branch rename --remote warns about the pull request", "pull request")
	r.check("rename --remote: the new branch is on origin", r.hasRef(ghBare, "refs/heads/fix/remote-new"), "the new branch is not on origin")
	r.check("rename --remote: the old branch is gone from origin", !r.hasRef(ghBare, "refs/heads/fix/remote-old"), "the old branch is still on origin")
	// On a forgejo host, --remote uses the rename call, so the pull request keeps its head.
	r.run(FG, "branch", "create", "fix/api-old", "main")
	r.git(FG, "push", "-q", "origin", "fix/api-old")
	r.renames = nil
	r.run(FG, "branch", "rename", "--remote", "fix/api-old", "fix/api-new")
	r.wantRC("branch rename --remote on forgejo", 0)
	r.wantOut("branch rename --remote on forgejo keeps the pull request", "keeps its number")
	r.check("rename --remote on forgejo: a rename call was made [deviation: the fake driver got the call, not tea]",
		strings.Join(r.renames, "|") == "fix/api-old fix/api-new", strings.Join(r.renames, "|"))
	r.check("rename --remote on forgejo: the new branch is on origin", r.hasRef(fgBare, "refs/heads/fix/api-new"), "the new branch is not on origin")
	r.check("rename --remote on forgejo: the old branch is gone from origin", !r.hasRef(fgBare, "refs/heads/fix/api-old"), "the old branch is still on origin")
	cur := r.current(FG)
	r.check("rename --remote on forgejo: on the new branch", cur == "fix/api-new", cur)
	r.run(FG, "branch", "switch", "main")
	r.run(FG, "branch", "delete", "fix/api-new")
	r.run(GH, "branch", "switch", "main")
	r.wantRC("branch switch", 0)
	r.run(GH, "branch", "delete", "fix/remote-new")
	r.wantRC("branch delete of the renamed branch", 0)
	r.run(GH, "branch", "delete", "fix/renamed-change")
	r.wantRC("branch delete of a merged branch", 0)
	r.run(GH, "branch", "create", "fix/second-change", "main")
	r.wantRC("branch create from a start point", 0)

	// --- vcs git commit ------------------------------------------------------
	// GH is on fix/second-change. FG is on main, and its branch fix/first-change stays.
	r.git(FG, "switch", "-q", "fix/first-change")
	r.appendTo(filepath.Join(GH, "file.txt"), "two\n")
	r.git(GH, "add", "-A")
	before := r.head(GH)
	r.run(GH, "commit", "-m", "Utilize the file")
	r.wantRC("commit prose finding", 65)
	r.wantOut("commit prose finding [deviation: the text is 'finding(s)']", "finding(s)")
	r.wantOut("commit prose finding names a re-check [deviation: no lint-prose.sh. The hint says to rewrite the message and run again]", "Rewrite the commit message")
	r.check("commit prose finding: no commit happened", r.head(GH) == before, "a commit happened")
	r.run(GH, "commit", "-m", "Add a line", "-m", "Co-Authored-By: Claude <noreply@anthropic.com>")
	r.wantRC("commit attribution", 65)
	r.wantOut("commit attribution says what to delete", "Delete that line from the commit message")
	r.run(GH, "commit", "--no-verify", "-m", "Add a line")
	r.wantRC("commit --no-verify", 64)
	r.wantOut("commit --no-verify says how to run a hook", "git hook run pre-commit")
	r.run(GH, "commit", "-nm", "Add a line")
	r.wantRC("commit -nm", 64)
	r.run(GH, "commit", "-n", "-m", "Add a line")
	r.wantRC("commit -n", 64)
	r.wantOut("commit -n says how to run a hook", "git hook run pre-commit")
	r.write(filepath.Join(GH, "WIRING_BAD"), "", 0o644)
	r.run(GH, "commit", "-m", "Add a line")
	r.wantRC("commit with broken wiring", 1)
	r.wantOut("commit with broken wiring", "wiring is broken")
	r.wantOut("commit with broken wiring names the installer", "bin/install.sh")
	r.wantOut("commit with broken wiring names the check", "hooks/check-wiring.sh --ci")
	r.check("commit with broken wiring: no commit happened", r.head(GH) == before, "a commit happened")
	os.Remove(filepath.Join(GH, "WIRING_BAD"))
	r.write(filepath.Join(GH, "scripts", "vcs-checks.sh"), "#!/usr/bin/env bash\necho \"project check $1 runs\"\n[ -f \"$PWD/PROJECT_BAD\" ] && { echo \"project finding\"; exit 1; }\nexit 0\n", 0o755)
	r.write(filepath.Join(GH, "PROJECT_BAD"), "", 0o644)
	r.run(GH, "commit", "-m", "Add a line")
	r.wantRC("commit with a failing project check", 1)
	r.wantOut("commit with a failing project check", "project finding")
	r.wantOut("commit with a failing project check names the re-run", "./scripts/vcs-checks.sh commit")
	r.check("commit with a failing project check: no commit happened", r.head(GH) == before, "a commit happened")
	os.Remove(filepath.Join(GH, "PROJECT_BAD"))
	r.run(GH, "commit", "-m", "Add a line")
	r.wantRC("commit ok", 0)
	r.wantOut("commit ok", "check project: ok")
	r.wantOut("commit ok", "host=github")
	r.check("commit ok: a commit happened", r.head(GH) != before, "no commit happened")
	// A commit with nothing to commit fails, and the verb says to read the output.
	r.run(GH, "commit", "-m", "Add a line")
	r.wantRC("commit with nothing staged", 1)
	r.wantOut("commit with nothing staged", "Read the output above")
	// A message from a file.
	r.write(filepath.Join(r.tmp, "msg.txt"), "Utilize the file\n", 0o644)
	r.appendTo(filepath.Join(GH, "file.txt"), "three\n")
	r.git(GH, "add", "-A")
	r.run(GH, "commit", "-F", filepath.Join(r.tmp, "msg.txt"))
	r.wantRC("commit -F with a finding", 65)
	// -C <dir>
	r.run(r.tmp, "commit", "-C", GH, "-m", "Add a third line")
	r.wantRC("commit -C", 0)
	// On main.
	r.git(FG, "switch", "-q", "main")
	r.appendTo(filepath.Join(FG, "file.txt"), "two\n")
	r.git(FG, "add", "-A")
	r.run(FG, "commit", "-m", "Add a line")
	r.wantRC("commit on main", 1)
	r.wantOut("commit on main", "would land on main")
	r.git(FG, "switch", "-q", "fix/first-change")
	r.git(FG, "switch", "-q", "-c", "wip-thing")
	r.run(FG, "commit", "-m", "Add a line")
	r.wantRC("commit on a bad branch name", 1)
	r.wantOut("commit on a bad branch name [deviation: the guard reports first, as guard:]", "guard:")
	r.git(FG, "switch", "-q", "fix/first-change")
	r.run(FG, "commit", "-m", "Add a line")
	r.wantRC("commit on forgejo", 0)
	r.wantOut("commit on forgejo", "host=forgejo")

	// --- vcs git push --------------------------------------------------------
	for _, flag := range []string{"--force", "-f", "--force-with-lease", "--force-with-lease=x", "--force-if-includes", "-uf"} {
		r.run(GH, "push", flag, "origin", "fix/second-change")
		r.wantRC("push "+flag, 64)
	}
	r.run(GH, "push", "origin", "+fix/second-change")
	r.wantRC("push +refspec", 64)
	r.run(GH, "push", "--no-verify")
	r.wantRC("push --no-verify", 64)
	r.write(filepath.Join(GH, "WIRING_BAD"), "", 0o644)
	r.run(GH, "push")
	r.wantRC("push with broken wiring", 1)
	os.Remove(filepath.Join(GH, "WIRING_BAD"))
	r.run(GH, "push")
	r.wantRC("push on github", 0)
	r.wantOut("push on github", "host=github")
	r.wantOut("push on github [deviation: the hint says vcs pr create, not gh pr create]", "vcs pr create --head fix/second-change")
	r.wantOut("push on github", "https://github.com/acme/widgets/compare/main...fix/second-change?expand=1")
	r.check("push on github: the branch is on the remote", r.hasRef(ghBare, "refs/heads/fix/second-change"), "the branch is not on the remote")
	r.run(FG, "push")
	r.wantRC("push on forgejo", 0)
	r.wantOut("push on forgejo", "host=forgejo")
	r.wantOut("push on forgejo [deviation: the hint says vcs pr create, not vcs-pr.sh create]", "vcs pr create")
	r.wantOut("push on forgejo", "https://git.grey.ooo/Grey.ooo/widgets/compare/main...fix/first-change")
	r.noOut("push on forgejo", "gh pr create")
	// A second push has an upstream, so it needs no -u.
	r.appendTo(filepath.Join(FG, "file.txt"), "four\n")
	r.git(FG, "add", "-A")
	r.run(FG, "commit", "-m", "Add a fourth line")
	r.wantRC("commit before the second push", 0)
	r.run(FG, "push")
	r.wantRC("push with an upstream", 0)
	// A branch cut from origin/main tracks origin/main. The push must still work, and it must not
	// push to main.
	r.git(FG, "switch", "-q", "-c", "fix/tracks-main", "origin/main")
	r.appendTo(filepath.Join(FG, "file.txt"), "five\n")
	r.git(FG, "add", "-A")
	r.run(FG, "commit", "-m", "Add a fifth line")
	r.wantRC("commit on a branch that tracks main", 0)
	r.run(FG, "push")
	r.wantRC("push of a branch that tracks main", 0)
	r.check("push of a branch that tracks main: the branch is on the remote", r.hasRef(fgBare, "refs/heads/fix/tracks-main"), "the branch is not on the remote")
	r.check("push of a branch that tracks main: main did not move on the remote",
		r.git(fgBare, "rev-parse", "refs/heads/main") == r.git(FG, "rev-parse", "main"), "main moved on the remote")
	r.git(FG, "switch", "-q", "fix/first-change")

	// --- vcs git merge -------------------------------------------------------
	// Make main move, then merge it into the feature branch.
	r.git(GH, "switch", "-q", "main")
	r.write(filepath.Join(GH, "main.txt"), "main change\n", 0o644)
	r.git(GH, "add", "-A")
	r.commitRaw(GH, "Change main")
	r.git(GH, "update-ref", "refs/remotes/origin/main", "main")
	r.git(GH, "switch", "-q", "fix/second-change")
	r.run(GH, "merge", "--no-verify", "origin/main")
	r.wantRC("merge --no-verify", 64)
	r.run(GH, "merge", "-m", "Utilize main", "origin/main")
	r.wantRC("merge message finding", 65)
	r.run(GH, "merge", "-m", "Merge main into the branch", "origin/main")
	r.wantRC("merge ok", 0)
	r.wantOut("merge ok", "host=github")
	r.run(GH, "merge")
	r.wantRC("merge with no arguments", 64)
	r.git(GH, "switch", "-q", "main")
	r.run(GH, "merge", "fix/second-change")
	r.wantRC("merge on main", 1)
	r.wantOut("merge on main", "would land on main")
	// A conflict is a non-zero exit, and --abort skips the checks.
	r.git(FG, "switch", "-q", "main")
	r.write(filepath.Join(FG, "clash.txt"), "left\n", 0o644)
	r.git(FG, "add", "-A")
	r.commitRaw(FG, "Add left")
	r.git(FG, "switch", "-q", "-c", "fix/clash-change", "main~1")
	r.write(filepath.Join(FG, "clash.txt"), "right\n", 0o644)
	r.git(FG, "add", "-A")
	r.commitRaw(FG, "Add right")
	r.run(FG, "merge", "main")
	r.wantRC("merge with a conflict", 1)
	r.wantOut("merge with a conflict", "CONFLICT")
	r.write(filepath.Join(FG, "WIRING_BAD"), "", 0o644)
	r.run(FG, "merge", "--abort")
	r.wantRC("merge --abort skips the checks", 0)
	os.Remove(filepath.Join(FG, "WIRING_BAD"))

	// The wiring check fails because the branch lacks links that main has gained. The merge of main
	// is the fix, so the verb runs it, then checks the wiring on the result.
	healRepo := func(name string) string { // a repo whose branch needs links/a and links/b, with links/a only
		d := r.newRepo(name, "https://git.grey.ooo/Grey.ooo/widgets.git")
		r.git(d, "switch", "-q", "-c", "fix/needs-links")
		r.write(filepath.Join(d, "wiring.need"), "links/a\nlinks/b\n", 0o644)
		r.write(filepath.Join(d, "links", "a"), "", 0o644)
		r.git(d, "add", "-A")
		r.commitRaw(d, "Need two links")
		return d
	}
	mainMove := func(d string, paths ...string) { // add the paths on main, as a new commit of origin/main
		r.git(d, "switch", "-q", "main")
		for _, p := range paths {
			r.write(filepath.Join(d, p), "", 0o644)
		}
		r.git(d, "add", "-A")
		r.commitRaw(d, "Main moves")
		r.git(d, "update-ref", "refs/remotes/origin/main", "main")
		r.git(d, "switch", "-q", "fix/needs-links")
	}

	// Main adds the missing link: the check fails before the merge and passes after it.
	HL := healRepo("heal")
	mainMove(HL, "links/b")
	before = r.head(HL)
	r.run(HL, "merge", "-m", "Merge main into the branch", "origin/main")
	r.wantRC("merge heals the wiring", 0)
	r.wantOut("merge heals the wiring", "check wiring (after the merge): ok")
	_, statErr := os.Stat(filepath.Join(HL, "links", "b"))
	r.check("merge heals the wiring: the merge landed", statErr == nil && r.head(HL) != before, "the merge did not land")
	r.run(HL, "merge", "-m", "Merge main again", "origin/main")
	r.wantRC("merge after the wiring holds", 0)
	r.wantOut("merge after the wiring holds", "check wiring: ok")

	// Main lacks the link too: the merge runs, the check still fails, and the branch goes back.
	HN := healRepo("heal-no")
	r.appendTo(filepath.Join(HN, "wiring.need"), "links/c\n")
	r.git(HN, "-c", "core.hooksPath=/dev/null", "commit", "-q", "-am", "Need a third link")
	mainMove(HN, "links/b")
	before = r.head(HN)
	r.run(HN, "merge", "-m", "Merge main into the branch", "origin/main")
	r.wantRC("merge refused when main lacks the links too", 1)
	r.wantOut("merge refused when main lacks the links too", "links/c is missing")
	r.wantOut("merge refused when main lacks the links too", "still fails after the merge")
	_, statErr = os.Stat(filepath.Join(HN, "links", "b"))
	r.check("merge refused when main lacks the links too: the branch did not move", r.head(HN) == before && statErr != nil, "the branch moved")

	// A dirty tree gives no room to try the merge, so the original refusal stands.
	HD := healRepo("heal-dirty")
	mainMove(HD, "links/b")
	r.appendTo(filepath.Join(HD, "links", "a"), "edit\n")
	r.run(HD, "merge", "-m", "Merge main into the branch", "origin/main")
	r.wantRC("merge refused on a dirty tree", 1)
	r.wantOut("merge refused on a dirty tree", "links/b is missing")
	r.wantOut("merge refused on a dirty tree [deviation: the text is \"can't try the merge\"]", "try the merge")
	_, statErr = os.Stat(filepath.Join(HD, "links", "b"))
	r.check("merge refused on a dirty tree: the merge did not run", statErr != nil, "the merge ran")

	// A conflict while the wiring fails: the merge stops, --continue refuses a failing result, and
	// --continue passes once the tree holds every link.
	HC := healRepo("heal-clash")
	r.write(filepath.Join(HC, "clash.txt"), "right\n", 0o644)
	r.git(HC, "add", "-A")
	r.commitRaw(HC, "Add right")
	r.git(HC, "switch", "-q", "main")
	r.write(filepath.Join(HC, "clash.txt"), "left\n", 0o644)
	r.write(filepath.Join(HC, "links", "b"), "", 0o644)
	r.git(HC, "add", "-A")
	r.commitRaw(HC, "Add left and a link")
	r.git(HC, "update-ref", "refs/remotes/origin/main", "main")
	r.git(HC, "switch", "-q", "fix/needs-links")
	r.run(HC, "merge", "-m", "Merge main into the branch", "origin/main")
	r.wantRC("merge with a conflict and a failing wiring", 1)
	r.wantOut("merge with a conflict and a failing wiring", "CONFLICT")
	r.write(filepath.Join(HC, "clash.txt"), "both\n", 0o644)
	r.git(HC, "add", "clash.txt")
	os.Remove(filepath.Join(HC, "links", "b"))
	r.run(HC, "merge", "--continue")
	r.wantRC("--continue refuses a failing wiring", 1)
	r.wantOut("--continue refuses a failing wiring", "links/b is missing")
	r.write(filepath.Join(HC, "links", "b"), "", 0o644)
	r.run(HC, "merge", "--continue")
	r.wantRC("--continue with the wiring holding", 0)

	// --- vcs git rebase ------------------------------------------------------
	r.run(GH, "branch", "create", "fix/rebase-me", "main")
	r.write(filepath.Join(GH, "r.txt"), "r\n", 0o644)
	r.git(GH, "add", "-A")
	r.commitRaw(GH, "Add r")
	r.run(GH, "rebase", "-i", "main")
	r.wantRC("rebase -i", 64)
	r.run(GH, "rebase", "--no-verify", "main")
	r.wantRC("rebase --no-verify", 64)
	r.run(GH, "rebase", "main")
	r.wantRC("rebase of an unpushed branch", 0)
	r.wantOut("rebase of an unpushed branch", "host=github")
	r.git(GH, "update-ref", "refs/remotes/origin/fix/rebase-me", "HEAD")
	r.run(GH, "rebase", "main")
	r.wantRC("rebase of a pushed branch", 1)
	r.wantOut("rebase of a pushed branch", "force-push")
	r.wantOut("rebase of a pushed branch [deviation: the hint says vcs git merge, not vcs-merge.sh]", "vcs git merge origin/main")
	r.git(GH, "switch", "-q", "main")
	r.run(GH, "rebase", "fix/second-change")
	r.wantRC("rebase on main", 1)

	if r.n != 145 {
		t.Errorf("the port runs %d checks, and the shell test runs 145", r.n)
	}

	// --- extra checks, not in the shell test ---------------------------------
	r.extra = true
	// A name that passes the guard but breaks the project's lint script reports the check by name.
	r.git(GH, "switch", "-q", "main")
	r.run(GH, "branch", "create", "chore/bad-for-the-stub")
	r.wantRC("branch create of a name that breaks the lint script", 1)
	r.wantOut("branch create of a name that breaks the lint script", "check branch-name failed")
	r.git(FG, "switch", "-q", "fix/first-change")
	r.git(FG, "switch", "-q", "-c", "chore/stub-fails")
	r.appendTo(filepath.Join(FG, "file.txt"), "six\n")
	r.git(FG, "add", "-A")
	r.run(FG, "commit", "-m", "Add a line")
	r.wantRC("commit on a name that breaks the lint script", 1)
	r.wantOut("commit on a name that breaks the lint script", "check branch-name failed")
	r.git(FG, "switch", "-q", "fix/first-change")
	// A repo with no origin gets no host note and no pull request hint, and no verb fails for the lack.
	NO := filepath.Join(r.tmp, "no-origin")
	r.git(r.tmp, "init", "-q", "-b", "main", NO)
	r.write(filepath.Join(NO, "file.txt"), "one\n", 0o644)
	r.git(NO, "add", "-A")
	r.commitRaw(NO, "Start the repo")
	r.run(NO, "branch", "create", "fix/no-origin")
	r.wantRC("no origin: branch create works", 0)
	r.noOut("no origin: branch create", "host=")
	r.appendTo(filepath.Join(NO, "file.txt"), "two\n")
	r.git(NO, "add", "-A")
	r.run(NO, "commit", "-m", "Add a line")
	r.wantRC("no origin: commit works", 0)
	r.noOut("no origin: commit", "host=")
	r.run(NO, "branch", "switch", "main")
	r.wantRC("no origin: branch switch works", 0)
	r.noOut("no origin: branch switch", "host=")
	r.git(NO, "switch", "-q", "fix/no-origin")
	r.run(NO, "merge", "main")
	r.wantRC("no origin: merge works", 0)
	r.noOut("no origin: merge", "host=")
	r.run(NO, "push")
	r.check("no origin: push fails in git, not in vcs", r.rc != 0 && !strings.Contains(r.out, "panic"), r.out)
	r.noOut("no origin: push", "host=")
	r.noOut("no origin: push", "Open the pull request")
	r.t.Logf("ported checks: %d, extra checks: %d", r.n, r.nx)
}
