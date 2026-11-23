package cli

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

const (
	healLabel = "➕healing"
	healColor = "d73a4a"
	healDesc  = "A worker heals this pull request now"
)

var liveList = regexp.MustCompile(`^(none|[0-9]+(,[0-9]+)*)$`)

// healCheck checks the arguments of vcs pr heal. It returns -1 when they are good.
func healCheck(env Env, v Verb, o opts) int {
	if len(o.pos) == 0 {
		return usage(env, v)
	}
	switch o.pos[0] {
	case "ensure":
		if len(o.pos) != 1 || o.flags["live"] != "" {
			return usage(env, v)
		}
	case "add", "remove":
		if len(o.pos) != 2 || !isNum(o.pos[1]) || o.flags["live"] != "" {
			return usage(env, v)
		}
	case "sweep":
		if len(o.pos) != 1 || !liveList.MatchString(o.flags["live"]) {
			return usage(env, v)
		}
	default:
		return usage(env, v)
	}
	return -1
}

// heal marks and unmarks the pull requests that a worker heals now. It replaces vcs-heal-label.sh.
// The label name comes from VCS_HEAL_LABEL, and it defaults to ➕healing.
func heal(ctx context.Context, c forge.Client, env Env, o opts) int {
	lm, ok1 := c.(forge.LabelManager)
	ed, ok2 := c.(forge.IssueEditor)
	rd, ok3 := c.(forge.Reader)
	if !ok1 || !ok2 || !ok3 {
		fmt.Fprintln(env.Stderr, "vcs: this host has no label, issue edit, or pr list support yet")
		return ExitUnavailable
	}
	name := healLabel
	if e := env.Getenv("VCS_HEAL_LABEL"); e != "" {
		name = e
	}
	ensure := func() (string, error) {
		ls, err := lm.LabelList(ctx)
		if err != nil {
			return "", err
		}
		for _, l := range ls {
			if l.Name == name {
				return "exists", nil
			}
		}
		return "created", lm.LabelCreate(ctx, name, healColor, healDesc)
	}
	switch o.pos[0] {
	case "ensure":
		r, err := ensure()
		if err != nil {
			return fail(env, err)
		}
		row(env, r)
	case "add":
		if _, err := ensure(); err != nil {
			return fail(env, err)
		}
		n, _ := strconv.Atoi(o.pos[1])
		if err := ed.IssueEdit(ctx, n, forge.IssueEdit{AddLabels: []string{name}}); err != nil {
			return fail(env, err)
		}
		row(env, "ok")
	case "remove":
		n, _ := strconv.Atoi(o.pos[1])
		marked, err := rd.PRList(ctx, forge.ListFilter{State: "all", Labels: []string{name}})
		if err != nil {
			return fail(env, err)
		}
		found := false
		for _, p := range marked {
			found = found || p.Number == n
		}
		if !found {
			row(env, "absent")
			return ExitOK
		}
		if err := ed.IssueEdit(ctx, n, forge.IssueEdit{RmLabels: []string{name}}); err != nil {
			return fail(env, err)
		}
		row(env, "ok")
	case "sweep":
		marked, err := rd.PRList(ctx, forge.ListFilter{State: "all", Labels: []string{name}})
		if err != nil {
			return fail(env, err)
		}
		open, err := rd.PRList(ctx, forge.ListFilter{State: "open", Labels: []string{name}})
		if err != nil {
			return fail(env, err)
		}
		isOpen := map[int]bool{}
		for _, p := range open {
			isOpen[p.Number] = true
		}
		live := map[string]bool{}
		for _, s := range splitList(o.flags["live"]) {
			live[s] = true
		}
		for _, p := range marked {
			reason := ""
			switch {
			case !isOpen[p.Number]:
				reason = "closed"
			case !live[strconv.Itoa(p.Number)]:
				reason = "no-worker"
			default:
				s, err := c.PRStatus(ctx, p.Number)
				if err != nil {
					return fail(env, err)
				}
				if s.CI == "success" && p.Mergeable {
					reason = "healthy"
				}
			}
			if reason == "" {
				continue
			}
			if err := ed.IssueEdit(ctx, p.Number, forge.IssueEdit{RmLabels: []string{name}}); err != nil {
				return fail(env, err)
			}
			row(env, "cleared", strconv.Itoa(p.Number), reason)
		}
	}
	return ExitOK
}
