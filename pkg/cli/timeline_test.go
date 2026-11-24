package cli

import (
	"context"
	"errors"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// timeliner is a fake that also lists the timeline of an issue.
type timeliner struct {
	fake
	rows []forge.TimelineEvent
	n    *int
}

func (l timeliner) IssueTimeline(_ context.Context, n int) ([]forge.TimelineEvent, error) {
	if l.n != nil {
		*l.n = n
	}
	return l.rows, l.err
}

func TestIssueTimeline(t *testing.T) {
	var n int
	c := timeliner{n: &n, rows: []forge.TimelineEvent{
		{ID: 1, Created: "2026-11-23T09:12:00Z", Type: "label", User: "matt", Detail: "+in-progress"},
		{ID: 2, Created: "2026-11-23T09:20:00Z", Type: "comment", User: "bot", Detail: "a\\nb"},
	}}
	code, out, _ := run(t, c, nil, true, "issue", "timeline", "14")
	want := "1\t2026-11-23T09:12:00Z\tlabel\tmatt\t+in-progress\n2\t2026-11-23T09:20:00Z\tcomment\tbot\ta\\nb\n"
	if code != ExitOK || out != want || n != 14 {
		t.Errorf("code=%d out=%q n=%d", code, out, n)
	}
	if code, out, _ = run(t, timeliner{}, nil, true, "issue", "timeline", "14"); code != ExitOK || out != "" {
		t.Errorf("empty: code=%d out=%q", code, out)
	}
}

func TestIssueTimelineErrors(t *testing.T) {
	for _, args := range [][]string{{"issue", "timeline"}, {"issue", "timeline", "x"}, {"issue", "timeline", "1", "2"}} {
		if code, _, _ := run(t, timeliner{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want usage", args, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, true, "issue", "timeline", "1"); code != ExitUnavailable {
		t.Errorf("no support: code=%d", code)
	}
	if code, _, _ := run(t, timeliner{fake: fake{err: errors.New("boom")}}, nil, true, "issue", "timeline", "1"); code != ExitFail {
		t.Errorf("fail: code=%d", code)
	}
}
