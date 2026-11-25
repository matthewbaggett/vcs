package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type noRead struct{}

func (noRead) Read([]byte) (int, error) { return 0, errors.New("help read the input") }

func runHelp(tty bool, stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	e := Env{Stdout: &out, Stderr: &errb, IsTerminal: tty, Getenv: func(string) string { return "" }}
	if stdin == "!" {
		e.Stdin = noRead{}
	} else {
		e.Stdin = strings.NewReader(stdin)
	}
	return Run(context.Background(), args, e), out.String(), errb.String()
}

func TestEveryNounHasAPurpose(t *testing.T) {
	for _, n := range nouns() {
		if NounPurpose[n] == "" {
			t.Errorf("noun %s has no purpose line", n)
		}
	}
}

func TestHelpPlainOutsideATerminal(t *testing.T) {
	// The input fails on a read, so a prompt would show as an error.
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		code, out, _ := runHelp(false, "!", args...)
		if code != 0 || !strings.Contains(out, "usage: vcs <noun> <verb>") || !strings.Contains(out, "  pr ") || strings.Contains(out, "Pick a") {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

func TestHelpNounAndVerbPages(t *testing.T) {
	code, out, _ := runHelp(true, "!", "help", "ci")
	if code != 0 || !strings.Contains(out, "ci rerun") || !strings.Contains(out, "ci artifact get") {
		t.Errorf("noun: %d %q", code, out)
	}
	code, out, _ = runHelp(false, "!", "help", "ci", "rerun")
	if code != 0 || !strings.Contains(out, "usage: ") || !strings.Contains(out, "needs a web session. Run: vcs session login") || !strings.Contains(out, "Example:") {
		t.Errorf("verb: %d %q", code, out)
	}
	if code, out, _ = runHelp(false, "", "help", "ci", "artifact", "get"); code != 0 || !strings.Contains(out, "usage: ") {
		t.Errorf("three words: %d %q", code, out)
	}
	if code, out, _ = runHelp(false, "", "help", "pr", "status"); strings.Contains(out, "web session") {
		t.Errorf("pr status needs no session: %q", out)
	}
	if code, _, errs := runHelp(false, "", "help", "nope"); code != ExitUsage || !strings.Contains(errs, "no noun") {
		t.Errorf("bad noun: %d %q", code, errs)
	}
	if code, _, errs := runHelp(false, "", "help", "pr", "nope"); code != ExitUsage || !strings.Contains(errs, "no verb") {
		t.Errorf("bad verb: %d %q", code, errs)
	}
	if code, _, _ := runHelp(false, "", "help", "a", "b", "c", "d"); code != ExitUsage {
		t.Errorf("four words: %d", code)
	}
}

func TestHelpMenuInATerminal(t *testing.T) {
	// A name picks the noun, a number picks the verb, a bad entry asks again.
	code, out, _ := runHelp(true, "pr\nzzz\n1\n", "help")
	first := verbsOf("pr")[0]
	if code != 0 || !strings.Contains(out, "Nouns:") || !strings.Contains(out, "Verbs of pr:") ||
		!strings.Contains(out, "No such choice: zzz") || !strings.Contains(out, "usage: "+first.Usage) {
		t.Errorf("menu: %d %q", code, out)
	}
	// A noun with one verb goes straight to its page.
	if code, out, _ = runHelp(true, "version\n", "help"); code != 0 || !strings.Contains(out, "usage: vcs version") {
		t.Errorf("one verb: %d %q", code, out)
	}
	// q and the end of the input quit with no page.
	for _, in := range []string{"q\n", ""} {
		if code, out, _ = runHelp(true, in, "help"); code != 0 || strings.Contains(out, "usage: ") {
			t.Errorf("quit %q: %d %q", in, code, out)
		}
	}
}
