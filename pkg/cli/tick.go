package cli

import (
	"context"
	"fmt"
	"strconv"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// tick reads every open pull request once. It merges each one that is ready, and it reports each
// one that needs a person. It replaces vcs-tick-merge.sh.
//
// A pull request is ready when it isn't a draft, it merges clean, it targets main, and its
// checks passed. After a merge, tick reads the pull request again. The state must say merged.
func tick(ctx context.Context, c forge.Client, env Env, dry bool) int {
	rd, ok := c.(forge.Reader)
	if !ok {
		fmt.Fprintln(env.Stderr, "vcs: this host has no pr list yet")
		return ExitUnavailable
	}
	prs, err := rd.PRList(ctx, forge.ListFilter{State: "open"})
	if err != nil {
		return fail(env, err)
	}
	attention := false
	say := func(word string, n int, sha string, extra ...string) {
		attention = true
		row(env, append([]string{word, strconv.Itoa(n), sha}, extra...)...)
	}
	for _, p := range prs {
		s, err := c.PRStatus(ctx, p.Number)
		if err != nil || s.State != "open" {
			continue
		}
		switch {
		case s.Draft:
			switch s.CI {
			case "success":
				say("DRAFT-GREEN", p.Number, s.HeadSHA)
			case "failure":
				say("DRAFT-RED", p.Number, s.HeadSHA)
			}
			continue
		case s.CI == "failure":
			say("RED", p.Number, s.HeadSHA)
			continue
		case !s.Mergeable:
			say("NOT-MERGEABLE", p.Number, s.HeadSHA, "ci="+s.CI)
			continue
		case s.CI != "success" || p.Base != "main":
			continue
		case dry:
			say("WOULD-MERGE", p.Number, s.HeadSHA)
			continue
		}
		m, ok := c.(forge.PRMerger)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no pr merge yet")
			return ExitUnavailable
		}
		mergeErr := m.PRMerge(ctx, p.Number, forge.PRMerge{})
		after, readErr := c.PRStatus(ctx, p.Number)
		landed := readErr == nil && after.State == "merged"
		switch {
		case landed:
			say("MERGED", p.Number, s.HeadSHA)
		case mergeErr == nil:
			say("MERGED-BUT-NOT-LANDED", p.Number, s.HeadSHA)
		default:
			say("MERGE-FAILED", p.Number, s.HeadSHA)
		}
	}
	if !attention {
		fmt.Fprintln(env.Stdout, "QUIET")
	}
	return ExitOK
}
