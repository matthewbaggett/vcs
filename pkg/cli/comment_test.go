package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/lint"
)

// commenter is a fake that also manages comments.
type commenter struct {
	fake
	rows []forge.CommentRow
	last *string
}

func (m commenter) CommentList(_ context.Context, n int) ([]forge.CommentRow, error) {
	return m.rows, m.err
}
func (m commenter) CommentEdit(_ context.Context, id int64, body string) error {
	*m.last = fmt.Sprintf("edit %d|%s", id, unstamp(body))
	return m.err
}
func (m commenter) CommentDelete(_ context.Context, id int64) error {
	*m.last = fmt.Sprintf("delete %d", id)
	return m.err
}

func TestCommentVerbs(t *testing.T) {
	var last string
	c := commenter{
		rows: []forge.CommentRow{{ID: 501, Author: "maria", Created: "c1", Updated: "u1", Body: "One.\r\nTwo\tthree"}},
		last: &last,
	}
	code, out, _ := run(t, c, nil, true, "comment", "list", "14")
	if code != 0 || out != "501\tmaria\tc1\tOne.\\nTwo three\tu1\n" {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
	code, out, _ = run(t, c, nil, true, "comment", "edit", "501", "--body", "Fixed now.")
	if code != 0 || out != "ok\n" || last != "edit 501|Fixed now." {
		t.Fatalf("edit: code=%d out=%q last=%q", code, out, last)
	}
	code, out, _ = run(t, c, nil, true, "comment", "delete", "501")
	if code != 0 || out != "ok\n" || last != "delete 501" {
		t.Fatalf("delete: code=%d out=%q last=%q", code, out, last)
	}
}

func TestCommentUsage(t *testing.T) {
	var last string
	c := commenter{last: &last}
	for _, args := range [][]string{
		{"comment", "list"},
		{"comment", "list", "x"},
		{"comment", "edit", "501"},
		{"comment", "edit", "501", "--body", "a", "--body-file", "b"},
		{"comment", "edit", "x", "--body", "a"},
		{"comment", "delete"},
		{"comment", "delete", "0"},
	} {
		if code, _, _ := run(t, c, nil, true, args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want usage", args, code)
		}
	}
	if last != "" {
		t.Errorf("a bad call sent %q", last)
	}
}

func TestCommentEditLintStopsTheSend(t *testing.T) {
	var last string
	c := commenter{last: &last}
	var out, errb bytes.Buffer
	env := Env{
		Stdout: &out, Stderr: &errb,
		Getenv:    func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return c, nil },
		Checkers:  []lint.Checker{stubChecker{fs: []lint.Finding{{Checker: "stub", Line: 1, Column: 1, Rule: "r", Message: "bad word"}}}},
	}
	code := Run(context.Background(), []string{"comment", "edit", "501", "--body", "x"}, env)
	if code != ExitLint || last != "" || !strings.Contains(errb.String(), "bad word") {
		t.Fatalf("code=%d last=%q err=%q", code, last, errb.String())
	}
}

func TestCommentNoSupport(t *testing.T) {
	code, _, errOut := run(t, fake{}, nil, true, "comment", "list", "14")
	if code != ExitUnavailable || !strings.Contains(errOut, "no comment support") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}
