package cli

import (
	"context"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type wfFake struct {
	fake
	file, ref string
}

func (f *wfFake) WorkflowDispatch(_ context.Context, file, ref string) error {
	f.file, f.ref = file, ref
	return f.err
}

func TestWorkflowDispatch(t *testing.T) {
	c := &wfFake{}
	if code, out, _ := run(t, c, nil, false, "workflow", "dispatch", "nightly.yml", "--ref", "release/1.0"); code != ExitOK || out != "ok\n" || c.file != "nightly.yml" || c.ref != "release/1.0" {
		t.Errorf("dispatch: %d %q %+v", code, out, c)
	}
	for _, a := range [][]string{
		{"workflow", "dispatch", "nightly.yml"},
		{"workflow", "dispatch", "--ref", "main"},
		{"workflow", "dispatch", "nightly.txt", "--ref", "main"},
		{"workflow", "dispatch", "a.yml", "b.yml", "--ref", "main"},
		{"workflow", "dispatch", "a.yml", "--ref", "a..b"},
		{"workflow", "dispatch", "a.yml", "--ref", "a b"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	c.err = forge.ErrNotFound
	if code, _, _ := run(t, c, nil, false, "workflow", "dispatch", "a.yaml", "--ref", "main"); code != ExitFail {
		t.Errorf("missing: %d", code)
	}
	if code, _, _ := run(t, fake{}, nil, false, "workflow", "dispatch", "a.yml", "--ref", "main"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
	if refOK("a\x7fb") || refOK("") {
		t.Errorf("refOK accepts a bad ref")
	}
}
