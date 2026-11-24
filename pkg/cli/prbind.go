package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// prBind finds the pull request of a branch and prints what a session needs to start work on it.
// It reads only. A "pr" row gives the number and the columns of "pr status". A "red" row gives
// the columns of "ci failed" for each red job. The branch comes from --branch or from the current
// branch of the checkout. An index skips the search.
func prBind(ctx context.Context, c forge.Client, env Env, o opts) int {
	n := 0
	if len(o.pos) == 1 {
		n, _ = strconv.Atoi(o.pos[0])
	} else {
		branch := o.flags["branch"]
		if branch == "" {
			if env.Exec == nil {
				fmt.Fprintln(env.Stderr, "vcs: no current branch. Give a number or --branch <name>")
				return ExitFail
			}
			var out bytes.Buffer
			env.Exec(ctx, "git", []string{"branch", "--show-current"}, &out, io.Discard)
			branch = strings.TrimSpace(out.String())
		}
		if branch == "" {
			fmt.Fprintln(env.Stderr, "vcs: no current branch. Give a number or --branch <name>")
			return ExitFail
		}
		rd, ok := c.(forge.Reader)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no pr list yet")
			return ExitUnavailable
		}
		prs, err := rd.PRList(ctx, forge.ListFilter{State: "open"})
		if err != nil {
			return fail(env, err)
		}
		found := 0
		for _, p := range prs {
			if p.Head == branch {
				found++
				if p.Number > n {
					n = p.Number
				}
			}
		}
		if found == 0 {
			fmt.Fprintf(env.Stderr, "vcs: no open pull request has the head branch %s\n", branch)
			return ExitFail
		}
		if found > 1 {
			fmt.Fprintf(env.Stderr, "vcs: more than one open pull request has the head branch %s. Took the newest, %d\n", branch, n)
		}
	}
	s, err := c.PRStatus(ctx, n)
	if err != nil {
		return fail(env, err)
	}
	row(env, "pr", strconv.Itoa(n), s.State, strconv.FormatBool(s.Draft), strconv.FormatBool(s.Mergeable),
		strings.Join(s.ConflictedFiles, ","), s.CI, s.HeadSHA, s.URL)
	var red bytes.Buffer
	sub := env
	sub.Stdout = &red
	if code := ciFailed(ctx, c, sub, strconv.Itoa(n)); code != ExitOK {
		return code
	}
	for _, line := range strings.Split(strings.TrimRight(red.String(), "\n"), "\n") {
		if line != "" {
			fmt.Fprintf(env.Stdout, "red\t%s\n", line)
		}
	}
	sha := s.HeadSHA
	if len(sha) > 7 {
		sha = sha[:7]
	}
	fmt.Fprintf(env.Stderr, "vcs: bound pull request %d at %s. To wait on it, run vcs pr watch %d\n", n, sha, n)
	return ExitOK
}
