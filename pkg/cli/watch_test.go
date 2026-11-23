package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// seq returns each status in turn, then repeats the last one.
type seq struct {
	fake
	states []forge.PRStatus
	calls  *int
}

func (s seq) PRStatus(context.Context, int) (forge.PRStatus, error) {
	i := *s.calls
	*s.calls++
	if i >= len(s.states) {
		i = len(s.states) - 1
	}
	return s.states[i], nil
}

func watchRun(t *testing.T, states []forge.PRStatus, args ...string) (code int, out string, sleeps int) {
	t.Helper()
	var o, e bytes.Buffer
	calls := 0
	env := Env{
		Stdout: &o, Stderr: &e, IsTerminal: true,
		Getenv:    func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return seq{states: states, calls: &calls}, nil },
		Sleep:     func(context.Context, time.Duration) { sleeps++ },
	}
	code = Run(context.Background(), append([]string{"pr", "watch", "7"}, args...), env)
	return code, o.String(), sleeps
}

func TestWatchWaitsForGreen(t *testing.T) {
	p := forge.PRStatus{State: "open", CI: "pending", HeadSHA: "a", URL: "u"}
	g := forge.PRStatus{State: "open", CI: "success", Mergeable: true, HeadSHA: "a", URL: "u"}
	code, out, sleeps := watchRun(t, []forge.PRStatus{p, p, g})
	if code != ExitOK || sleeps != 2 {
		t.Fatalf("code=%d sleeps=%d", code, sleeps)
	}
	if want := "open\tfalse\tfalse\t\tpending\ta\tu\nopen\tfalse\ttrue\t\tsuccess\ta\tu\n"; out != want {
		t.Fatalf("out=%q", out)
	}
}

func TestWatchBadEnds(t *testing.T) {
	cases := map[string]forge.PRStatus{
		"failed check": {State: "open", CI: "failure"},
		"conflict":     {State: "open", CI: "pending", ConflictedFiles: []string{"a"}},
		"closed":       {State: "closed", CI: "success"},
	}
	for name, st := range cases {
		if code, _, _ := watchRun(t, []forge.PRStatus{st}); code != ExitFail {
			t.Errorf("%s: code=%d", name, code)
		}
	}
}

func TestWatchMergedAndDraft(t *testing.T) {
	if code, _, _ := watchRun(t, []forge.PRStatus{{State: "merged"}}); code != ExitOK {
		t.Errorf("merged: code=%d", code)
	}
	if code, _, _ := watchRun(t, []forge.PRStatus{{State: "open", Draft: true, CI: "success"}}); code != ExitOK {
		t.Errorf("draft: code=%d", code)
	}
}

func TestWatchTimeout(t *testing.T) {
	code, _, sleeps := watchRun(t, []forge.PRStatus{{State: "open", CI: "pending"}}, "--interval", "10", "--timeout", "30")
	if code != ExitFail || sleeps != 3 {
		t.Fatalf("code=%d sleeps=%d", code, sleeps)
	}
}

func TestWatchUsage(t *testing.T) {
	for _, a := range [][]string{{"--interval", "0"}, {"--timeout", "x"}} {
		if code, _, _ := watchRun(t, []forge.PRStatus{{State: "open"}}, a...); code != ExitUsage {
			t.Errorf("%v: code=%d", a, code)
		}
	}
}
