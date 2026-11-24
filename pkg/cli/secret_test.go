package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type secretFake struct {
	fake
	list    []forge.Secret
	setName string
	setVal  string
	sets    int
	deleted string
}

func (f *secretFake) SecretList(context.Context) ([]forge.Secret, error) { return f.list, f.err }
func (f *secretFake) SecretSet(_ context.Context, n, v string) error {
	f.sets++
	f.setName, f.setVal = n, v
	return f.err
}
func (f *secretFake) SecretDelete(_ context.Context, n string) error {
	f.deleted = n
	return f.err
}

func runIn(t *testing.T, c forge.Client, env map[string]string, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	e := Env{
		Stdout: &out, Stderr: &errb, Stdin: strings.NewReader(stdin),
		Getenv:    func(k string) string { return env[k] },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(host, repo string) (forge.Client, error) { return c, nil },
	}
	code := Run(context.Background(), args, e)
	return code, out.String(), errb.String()
}

func TestSecretList(t *testing.T) {
	c := &secretFake{list: []forge.Secret{{Name: "A", Created: "t1"}, {Name: "B", Created: "t2"}}}
	if code, out, _ := run(t, c, nil, false, "secret", "list"); code != ExitOK || out != "A\tt1\nB\tt2\n" {
		t.Errorf("list: %d %q", code, out)
	}
	if code, _, _ := run(t, c, nil, false, "secret", "list", "x"); code != ExitUsage {
		t.Errorf("usage: %d", code)
	}
	c.err = forge.ErrNoSession
	if code, _, _ := run(t, c, nil, false, "secret", "list"); code != ExitNoSession {
		t.Errorf("refused: %d", code)
	}
	if code, _, _ := run(t, fake{}, nil, false, "secret", "list"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
}

func TestSecretSet(t *testing.T) {
	c := &secretFake{}
	if code, out, errb := runIn(t, c, nil, "s3cret\n\n", "secret", "set", "KEY_1", "--value-file", "-"); code != ExitOK || out != "ok\n" || c.setName != "KEY_1" || c.setVal != "s3cret" || strings.Contains(out+errb, "s3cret") {
		t.Errorf("stdin: %d %q %q %+v", code, out, errb, c)
	}
	if code, _, _ := runIn(t, c, map[string]string{"TOK": "fromenv"}, "", "secret", "set", "K", "--from-env", "TOK"); code != ExitOK || c.setVal != "fromenv" {
		t.Errorf("env: %d %q", code, c.setVal)
	}
	f := filepath.Join(t.TempDir(), "v")
	os.WriteFile(f, []byte("filed\r\n"), 0o600)
	if code, _, _ := runIn(t, c, nil, "", "secret", "set", "K", "--value-file", f); code != ExitOK || c.setVal != "filed" {
		t.Errorf("file: %d %q", code, c.setVal)
	}
	before := c.sets
	for name, args := range map[string][]string{
		"empty env":   {"secret", "set", "K", "--from-env", "NOPE"},
		"empty stdin": {"secret", "set", "K", "--value-file", "-"},
		"no source":   {"secret", "set", "K"},
		"both":        {"secret", "set", "K", "--value-file", "-", "--from-env", "X"},
		"bad name":    {"secret", "set", "1bad", "--from-env", "X"},
		"no name":     {"secret", "set", "--from-env", "X"},
		"missing":     {"secret", "set", "K", "--value-file", "/no/such/file"},
	} {
		if code, _, _ := runIn(t, c, map[string]string{"X": "v"}, "\n", args...); code != ExitUsage && name != "both" && name != "bad name" && name != "no name" {
			t.Errorf("%s: %d", name, code)
		} else if code != ExitUsage {
			t.Errorf("%s: %d", name, code)
		}
	}
	if c.sets != before {
		t.Errorf("a refused call sent a secret")
	}
	c.err = forge.ErrNoSession
	if code, _, _ := runIn(t, c, nil, "v", "secret", "set", "K", "--value-file", "-"); code != ExitNoSession {
		t.Errorf("refused: %d", code)
	}
	if code, _, _ := runIn(t, fake{}, nil, "v", "secret", "set", "K", "--value-file", "-"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
}

func TestSecretDelete(t *testing.T) {
	c := &secretFake{}
	if code, out, _ := run(t, c, nil, false, "secret", "delete", "K"); code != ExitOK || out != "ok\n" || c.deleted != "K" {
		t.Errorf("delete: %d %q %q", code, out, c.deleted)
	}
	for _, a := range [][]string{{"secret", "delete"}, {"secret", "delete", "a", "b"}, {"secret", "delete", "a-b"}} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, false, "secret", "delete", "K"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
}
