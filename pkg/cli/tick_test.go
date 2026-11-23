package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// ticker serves one status for each pull request. A merge flips the state to merged, unless the
// number is in stuck. A number in refuse makes the merge fail.
type ticker struct {
	reader
	status map[int]forge.PRStatus
	merged *[]int
	stuck  map[int]bool
	refuse map[int]bool
}

func (t ticker) PRStatus(_ context.Context, n int) (forge.PRStatus, error) {
	s, ok := t.status[n]
	if !ok {
		return forge.PRStatus{}, forge.ErrNotFound
	}
	for _, m := range *t.merged {
		if m == n && !t.stuck[n] {
			s.State = "merged"
		}
	}
	return s, nil
}

func (t ticker) PRMerge(_ context.Context, n int, _ forge.PRMerge) error {
	if t.refuse[n] {
		return errors.New("refused")
	}
	*t.merged = append(*t.merged, n)
	return nil
}

func tickFixture() (ticker, *[]int) {
	open := func(ci string, mergeable bool) forge.PRStatus {
		return forge.PRStatus{State: "open", Mergeable: mergeable, CI: ci, HeadSHA: "abc1234"}
	}
	draft := func(ci string) forge.PRStatus { s := open(ci, true); s.Draft = true; return s }
	var merged []int
	t := ticker{
		reader: reader{prs: []forge.PRInfo{
			{Number: 1, Base: "main"},  // ready
			{Number: 2, Base: "main"},  // red
			{Number: 3, Base: "main"},  // conflict
			{Number: 4, Base: "main"},  // draft, green
			{Number: 5, Base: "main"},  // draft, red
			{Number: 6, Base: "dev"},   // other base, green
			{Number: 7, Base: "main"},  // pending
			{Number: 8, Base: "main"},  // merge refused
			{Number: 9, Base: "main"},  // merge returns, state stays open
			{Number: 10, Base: "main"}, // unreadable status
		}},
		status: map[int]forge.PRStatus{
			1: open("success", true), 2: open("failure", true), 3: open("pending", false),
			4: draft("success"), 5: draft("failure"), 6: open("success", true), 7: open("pending", true),
			8: open("success", true), 9: open("success", true),
		},
		merged: &merged,
		stuck:  map[int]bool{9: true},
		refuse: map[int]bool{8: true},
	}
	return t, &merged
}

func TestPRTick(t *testing.T) {
	c, merged := tickFixture()
	code, out, _ := run(t, c, nil, false, "pr", "tick")
	want := strings.Join([]string{
		"MERGED\t1\tabc1234",
		"RED\t2\tabc1234",
		"NOT-MERGEABLE\t3\tabc1234\tci=pending",
		"DRAFT-GREEN\t4\tabc1234",
		"DRAFT-RED\t5\tabc1234",
		"MERGE-FAILED\t8\tabc1234",
		"MERGED-BUT-NOT-LANDED\t9\tabc1234",
		"",
	}, "\n")
	if code != ExitOK || out != want {
		t.Errorf("tick: %d\n%q\nwant\n%q", code, out, want)
	}
	if len(*merged) != 2 || (*merged)[0] != 1 || (*merged)[1] != 9 {
		t.Errorf("merged %v, want [1 9]", *merged)
	}
}

func TestPRTickDryRunAndQuiet(t *testing.T) {
	c, merged := tickFixture()
	code, out, _ := run(t, c, nil, false, "pr", "tick", "--dry-run")
	if code != ExitOK || !strings.Contains(out, "WOULD-MERGE\t1\tabc1234\n") || !strings.Contains(out, "WOULD-MERGE\t8\t") || len(*merged) != 0 {
		t.Errorf("dry run: %d %q merged=%v", code, out, *merged)
	}
	quiet := ticker{reader: reader{prs: []forge.PRInfo{{Number: 6, Base: "dev"}, {Number: 7, Base: "main"}}},
		status: map[int]forge.PRStatus{
			6: {State: "open", Mergeable: true, CI: "success"},
			7: {State: "open", Mergeable: true, CI: "pending"},
		}, merged: &[]int{}}
	if code, out, _ := run(t, quiet, nil, false, "pr", "tick"); code != ExitOK || out != "QUIET\n" {
		t.Errorf("quiet: %d %q", code, out)
	}
}

func TestPRTickErrors(t *testing.T) {
	if code, _, _ := run(t, fake{}, nil, false, "pr", "tick"); code != ExitUnavailable {
		t.Errorf("no reader: %d", code)
	}
	if code, _, _ := run(t, reader{fake: fake{err: errors.New("boom")}}, nil, false, "pr", "tick"); code != ExitFail {
		t.Errorf("list failure: %d", code)
	}
	if code, _, _ := run(t, fake{}, nil, false, "pr", "tick", "5"); code != ExitUsage {
		t.Errorf("positional: %d", code)
	}
}
