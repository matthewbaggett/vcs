package cli

import (
	"context"
	"errors"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// newer is a fake that also lists the items above a mark.
type newer struct {
	fake
	rows  []forge.NewItem
	top   int
	mark  *int
	state *string
	any   *bool
}

func (l newer) IssuesNewer(_ context.Context, mark int, state string, anyKind bool) ([]forge.NewItem, error) {
	if l.mark != nil {
		*l.mark, *l.state, *l.any = mark, state, anyKind
	}
	return l.rows, l.err
}

func (l newer) TopIndex(context.Context) (int, error) { return l.top, l.err }

func TestIssueNewer(t *testing.T) {
	var mark int
	var state string
	var anyKind bool
	c := newer{mark: &mark, state: &state, any: &anyKind, top: 183, rows: []forge.NewItem{
		{Number: 183, Kind: "pr", Author: "matt", Title: "Add the board"},
		{Number: 181, Kind: "issue", Author: "bot", Title: "Fix"},
	}}
	code, out, _ := run(t, c, nil, true, "issue", "newer", "180", "--kind", "any", "--state", "all")
	want := "183\tpr\tmatt\tAdd the board\n181\tissue\tbot\tFix\n"
	if code != ExitOK || out != want || mark != 180 || state != "all" || !anyKind {
		t.Errorf("code=%d out=%q mark=%d state=%q any=%v", code, out, mark, state, anyKind)
	}
	run(t, c, nil, true, "issue", "newer", "0")
	if state != "open" || anyKind {
		t.Errorf("defaults: state=%q any=%v", state, anyKind)
	}
	if code, out, _ = run(t, c, nil, true, "issue", "top"); code != ExitOK || out != "183\n" {
		t.Errorf("top: code=%d out=%q", code, out)
	}
}

func TestIssueNewerErrors(t *testing.T) {
	for _, args := range [][]string{
		{"issue", "newer"}, {"issue", "newer", "x"}, {"issue", "newer", "1", "2"},
		{"issue", "newer", "1", "--state", "closed"}, {"issue", "newer", "1", "--kind", "pr"},
		{"issue", "top", "1"},
	} {
		if code, _, _ := run(t, newer{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want usage", args, code)
		}
	}
	for _, args := range [][]string{{"issue", "newer", "1"}, {"issue", "top"}} {
		if code, _, _ := run(t, fake{}, nil, true, args...); code != ExitUnavailable {
			t.Errorf("%v no support: code=%d", args, code)
		}
		if code, _, _ := run(t, newer{fake: fake{err: errors.New("boom")}}, nil, true, args...); code != ExitFail {
			t.Errorf("%v fail: code=%d", args, code)
		}
	}
}
