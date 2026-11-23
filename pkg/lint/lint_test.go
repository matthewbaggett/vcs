package lint

import (
	"errors"
	"strings"
	"testing"

	"git.grey.ooo/grey.ooo/lexi"
)

type stub struct {
	fs  []Finding
	err error
}

func (stub) Name() string                      { return "stub" }
func (s stub) Check(string) ([]Finding, error) { return s.fs, s.err }

func TestLexiFindsAndPasses(t *testing.T) {
	x, err := NewLexi(lexi.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fs, err := Run("He is a man's man.", x)
	if err != nil || len(fs) == 0 {
		t.Fatalf("want findings, got %v %v", fs, err)
	}
	if fs[0].Checker != "lexi" || fs[0].Line != 1 || fs[0].Rule == "" {
		t.Errorf("bad finding %+v", fs[0])
	}
	fs, err = Run("Run the check before each commit.", x)
	if err != nil || len(fs) != 0 {
		t.Fatalf("want none, got %v %v", fs, err)
	}
}

func TestRunOrdersByPosition(t *testing.T) {
	a := stub{fs: []Finding{{Checker: "a", Line: 3, Column: 1}, {Checker: "a", Line: 1, Column: 9}}}
	b := stub{fs: []Finding{{Checker: "b", Line: 1, Column: 2}}}
	fs, err := Run("x", a, b)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.Checker)
	}
	if strings.Join(got, "") != "baa" {
		t.Errorf("order %v", got)
	}
}

func TestRunStopsOnCheckerError(t *testing.T) {
	boom := errors.New("no binary")
	_, err := Run("x", stub{err: boom})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}
