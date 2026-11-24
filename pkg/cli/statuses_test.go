package cli

import (
	"context"
	"errors"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// lister is a fake that also lists commit statuses.
type lister struct {
	fake
	rows []forge.CommitStatus
	sha  *string
}

func (l lister) CommitStatuses(_ context.Context, sha string) ([]forge.CommitStatus, error) {
	if l.sha != nil {
		*l.sha = sha
	}
	return l.rows, l.err
}

func TestPRStatuses(t *testing.T) {
	var sha string
	c := lister{sha: &sha, rows: []forge.CommitStatus{
		{ID: 7, Context: "ci / test", Status: "failure", URL: "http://h/7", Description: "first"},
		{ID: 9, Context: "ci / test", Status: "success", URL: "http://h/9", Description: "second"},
		{ID: 8, Context: "ci / lint", Status: "pending", URL: ""},
	}}
	code, out, _ := run(t, c, nil, true, "pr", "statuses", "abc1234")
	want := "ci / test\tsuccess\thttp://h/9\nci / lint\tpending\t\nci / test\tfailure\thttp://h/7\n"
	if code != ExitOK || out != want || sha != "abc1234" {
		t.Errorf("all rows: code=%d out=%q sha=%q", code, out, sha)
	}
	code, out, _ = run(t, c, nil, true, "pr", "statuses", "abc1234", "--latest", "--description")
	want = "ci / lint\tpending\t\t\nci / test\tsuccess\thttp://h/9\tsecond\n"
	if code != ExitOK || out != want {
		t.Errorf("latest: code=%d out=%q", code, out)
	}
	if code, out, _ = run(t, lister{}, nil, true, "pr", "statuses", "abc1234"); code != ExitOK || out != "" {
		t.Errorf("empty list: code=%d out=%q", code, out)
	}
}

func TestPRStatusesErrors(t *testing.T) {
	for _, args := range [][]string{{"pr", "statuses"}, {"pr", "statuses", "a b"}, {"pr", "statuses", "a", "b"}} {
		if code, _, _ := run(t, lister{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want usage", args, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, true, "pr", "statuses", "abc1234"); code != ExitUnavailable {
		t.Errorf("no support: code=%d", code)
	}
	if code, _, _ := run(t, lister{fake: fake{err: errors.New("boom")}}, nil, true, "pr", "statuses", "abc1234"); code != ExitFail {
		t.Errorf("fail: code=%d", code)
	}
}
