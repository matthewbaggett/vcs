package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func bindEnv(t *testing.T, c forge.Client, branch string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	e := Env{
		Stdout: &out, Stderr: &errb,
		Getenv:    func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(host, repo string) (forge.Client, error) { return c, nil },
		Exec: func(_ context.Context, name string, a []string, stdout, _ io.Writer) int {
			io.WriteString(stdout, branch+"\n")
			return 0
		},
	}
	code := Run(context.Background(), args, e)
	return code, out.String(), errb.String()
}

func TestPRBind(t *testing.T) {
	c := reader{
		fake: fake{
			pr:   forge.PRStatus{State: "open", Draft: true, Mergeable: true, CI: "failure", HeadSHA: "abc1234def", URL: "u"},
			runs: []forge.Run{{ID: 9, Status: "failure", SHA: "abc1234def"}},
			jobs: []forge.Job{{ID: 7, Name: "gate", Status: "failure"}},
			log:  "::error::boom\n",
		},
		prs: []forge.PRInfo{{Number: 4, Head: "feature/x"}, {Number: 7, Head: "feature/x"}, {Number: 8, Head: "other"}},
	}
	want := "pr\t7\topen\ttrue\ttrue\t\tfailure\tabc1234def\tu\nred\t9\t7\tgate\tboom\n"
	code, out, errb := bindEnv(t, c, "feature/x", "pr", "bind")
	if code != ExitOK || out != want || !strings.Contains(errb, "Took the newest, 7") || !strings.Contains(errb, "vcs pr watch 7") {
		t.Fatalf("branch: %d %q %q", code, out, errb)
	}
	if code, out, _ := bindEnv(t, c, "", "pr", "bind", "--branch", "other"); code != ExitOK || !strings.HasPrefix(out, "pr\t8\t") {
		t.Errorf("--branch: %d %q", code, out)
	}
	if code, out, _ := bindEnv(t, c, "", "pr", "bind", "5"); code != ExitOK || !strings.HasPrefix(out, "pr\t5\t") {
		t.Errorf("index: %d %q", code, out)
	}
	if code, _, errb := bindEnv(t, c, "nothing", "pr", "bind"); code != ExitFail || !strings.Contains(errb, "no open pull request") {
		t.Errorf("none: %d %q", code, errb)
	}
	if code, _, _ := bindEnv(t, c, "", "pr", "bind"); code != ExitFail {
		t.Errorf("no branch: %d", code)
	}
	for _, a := range [][]string{{"pr", "bind", "x"}, {"pr", "bind", "5", "--branch", "b"}, {"pr", "bind", "1", "2"}} {
		if code, _, _ := bindEnv(t, c, "b", a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, _ := bindEnv(t, fake{}, "b", "pr", "bind"); code != ExitUnavailable {
		t.Errorf("no reader: %d", code)
	}
	c.err = forge.ErrNotFound
	if code, _, _ := bindEnv(t, c, "b", "pr", "bind"); code != ExitFail {
		t.Errorf("list fails: %d", code)
	}
	c.err = nil
	c.fake.err = forge.ErrNotFound
	c.prs = nil
	if code, _, _ := bindEnv(t, c, "", "pr", "bind", "5"); code != ExitFail {
		t.Errorf("status fails: %d", code)
	}
	// No Exec and no --branch: nothing to bind.
	var o2, e2 bytes.Buffer
	e := Env{Stdout: &o2, Stderr: &e2, Getenv: func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(host, repo string) (forge.Client, error) { return c, nil }}
	if code := Run(context.Background(), []string{"pr", "bind"}, e); code != ExitFail {
		t.Errorf("no exec: %d", code)
	}
}
