package cli

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// stater is a fake that also changes the state of a pull request.
type stater struct {
	fake
	last   *string
	ticked bool
	err    error
}

func (s stater) PRDraft(_ context.Context, n int) error {
	*s.last = fmt.Sprintf("draft %d", n)
	return s.err
}
func (s stater) PRClose(_ context.Context, n int) error {
	*s.last = fmt.Sprintf("close %d", n)
	return s.err
}
func (s stater) PRRebase(_ context.Context, n int) (bool, error) {
	*s.last = fmt.Sprintf("rebase %d", n)
	return s.ticked, s.err
}

func TestPRStateVerbs(t *testing.T) {
	var last string
	c := stater{last: &last, ticked: true}
	for _, tc := range []struct{ verb, out, sent string }{
		{"draft", "draft\t13\n", "draft 13"},
		{"close", "closed\t13\n", "close 13"},
		{"rebase", "ok\n", "rebase 13"},
	} {
		code, out, _ := run(t, c, nil, true, "pr", tc.verb, "13")
		if code != ExitOK || out != tc.out || last != tc.sent {
			t.Errorf("%s: code=%d out=%q last=%q", tc.verb, code, out, last)
		}
	}
	c.ticked = false
	if _, out, _ := run(t, c, nil, true, "pr", "rebase", "13"); out != "already\n" {
		t.Errorf("already: %q", out)
	}
}

func TestPRStateErrors(t *testing.T) {
	var last string
	for _, args := range [][]string{{"pr", "draft"}, {"pr", "close", "x"}, {"pr", "rebase", "0"}} {
		if code, _, _ := run(t, stater{last: &last}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want usage", args, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, true, "pr", "close", "13"); code != ExitUnavailable {
		t.Errorf("no support: code=%d", code)
	}
	if code, _, _ := run(t, stater{last: &last, err: forge.ErrUnsupported}, nil, true, "pr", "rebase", "13"); code != ExitUnavailable {
		t.Errorf("unsupported: code=%d", code)
	}
	if code, _, _ := run(t, stater{last: &last, err: errors.New("boom")}, nil, true, "pr", "draft", "13"); code != ExitFail {
		t.Errorf("fail: code=%d", code)
	}
}
