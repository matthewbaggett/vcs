package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	lexicli "git.grey.ooo/grey.ooo/lexi/cli"
)

type result struct {
	code        int
	out, errOut string
}

func runVCS(args []string, stdin string) result {
	var out, errOut bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errOut, Stdin: strings.NewReader(stdin)}
	code := Run(context.Background(), args, env)
	return result{code, out.String(), errOut.String()}
}

func runLexi(args []string, stdin string) result {
	var out, errOut bytes.Buffer
	code := lexicli.Run(args, strings.NewReader(stdin), &out, &errOut)
	return result{code, out.String(), errOut.String()}
}

// `vcs lexi` and `vcs alex` must match the lexi command line: the same stdout, stderr, and exit code.
func TestLexiAndAlexMatchTheLexiCLI(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"finding", []string{"--stdin"}, "His network looks good\n"},
		{"clean", []string{"--stdin"}, "The network looks good.\n"},
		{"plain text", []string{"--stdin", "-t"}, "He is a retard.\n"},
		{"json reporter", []string{"--stdin", "--reporter=json"}, "His network looks good\n"},
		{"quiet", []string{"--stdin", "-q"}, "His network looks good\n"},
		{"why", []string{"--stdin", "-w"}, "His network looks good\n"},
		{"help", []string{"--help"}, ""},
		{"version", []string{"--version"}, ""},
		{"missing file", []string{"no-such-file.md"}, ""},
	}
	for _, c := range cases {
		want := runLexi(c.args, c.stdin)
		for _, name := range []string{"lexi", "alex"} {
			got := runVCS(append([]string{name}, c.args...), c.stdin)
			if got != want {
				t.Errorf("%s: vcs %s differs from lexi\n got  %+v\n want %+v", c.name, name, got, want)
			}
		}
	}
}

// A finding proves the check ran, so the test does not pass on two empty results.
func TestLexiFindsSomething(t *testing.T) {
	r := runVCS([]string{"lexi", "--stdin"}, "His network looks good\n")
	if r.code == 0 && r.errOut == "" && r.out == "" {
		t.Fatalf("lexi found nothing: %+v", r)
	}
}
