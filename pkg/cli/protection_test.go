package cli

import (
	"context"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type protFake struct {
	fake
	rules []forge.Protection
}

func (f protFake) ProtectionList(context.Context) ([]forge.Protection, error) { return f.rules, f.err }

func TestProtectionList(t *testing.T) {
	c := protFake{rules: []forge.Protection{
		{Rule: "main", Contexts: []string{"CI / a (pull_request)", "b"}, Approvals: 1, Push: "none", Admins: true},
		{Rule: "release/*", Push: "whitelist"},
		{Rule: "deep/**", Push: "all"},
	}}
	all := "main\tCI / a (pull_request),b\t1\tnone\tyes\nrelease/*\t\t0\twhitelist\tno\ndeep/**\t\t0\tall\tno\n"
	if code, out, _ := run(t, c, nil, false, "protection", "list"); code != ExitOK || out != all {
		t.Errorf("all: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "protection", "list", "release/1.0"); code != ExitOK || out != "release/*\t\t0\twhitelist\tno\n" {
		t.Errorf("pattern: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "protection", "list", "release/a/b"); code != ExitOK || out != "" {
		t.Errorf("star stops at a slash: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "protection", "list", "deep/a/b"); code != ExitOK || out != "deep/**\t\t0\tall\tno\n" {
		t.Errorf("double star: %d %q", code, out)
	}
	for _, a := range [][]string{{"protection", "list", "a", "b"}, {"protection", "list", "bad name"}} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	c.err = forge.ErrNoSession
	if code, _, _ := run(t, c, nil, false, "protection", "list"); code != ExitNoSession {
		t.Errorf("refused: %d", code)
	}
	if code, _, _ := run(t, fake{}, nil, false, "protection", "list"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
	if protectionCovers("", "x") {
		t.Errorf("empty rule covers")
	}
}
