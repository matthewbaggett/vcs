package cli

import (
	"context"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type runnerFake struct {
	fake
	runners []forge.Runner
}

func (f runnerFake) RunnerList(context.Context) ([]forge.Runner, error) { return f.runners, f.err }

func TestRunnersList(t *testing.T) {
	c := runnerFake{runners: []forge.Runner{
		{ID: 3, Name: "build-1", Status: "idle", Labels: []string{"ubuntu-latest", "docker"}, UUID: "u-3", Version: "v6"},
		{ID: 4, Name: "build-2", Status: "offline", UUID: "u-4", Version: "v6"},
	}}
	want := "3\tbuild-1\tidle\tubuntu-latest,docker\tu-3\tv6\n4\tbuild-2\toffline\t\tu-4\tv6\n"
	if code, out, _ := run(t, c, nil, false, "runners", "list"); code != ExitOK || out != want {
		t.Errorf("list: %d %q", code, out)
	}
	c.runners = nil
	if code, out, _ := run(t, c, nil, false, "runners", "list"); code != ExitOK || out != "" {
		t.Errorf("none: %d %q", code, out)
	}
	c.err = forge.ErrNoSession
	if code, _, _ := run(t, c, nil, false, "runners", "list"); code != ExitNoSession {
		t.Errorf("refused: %d", code)
	}
	c.err = forge.ErrNotFound
	if code, _, _ := run(t, c, nil, false, "runners", "list"); code != ExitFail {
		t.Errorf("fail: %d", code)
	}
	if code, _, _ := run(t, c, nil, false, "runners", "list", "3"); code != ExitUsage {
		t.Errorf("usage: %d", code)
	}
	if code, _, errb := run(t, fake{}, nil, false, "runners", "list"); code != ExitUnavailable {
		t.Errorf("no driver call: %d %q", code, errb)
	}
}
