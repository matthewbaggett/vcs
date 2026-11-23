// Package lint checks the text that vcs sends. Each checker returns findings. A text with
// any finding does not go out.
package lint

import (
	"fmt"
	"sort"

	"git.grey.ooo/grey.ooo/lexi"
)

// Finding is one problem in a text. Line and column start at 1.
type Finding struct {
	Checker string
	Line    int
	Column  int
	Rule    string
	Message string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d:%d: %s (%s)", f.Checker, f.Line, f.Column, f.Message, f.Rule)
}

// Checker checks one text.
type Checker interface {
	Name() string
	Check(text string) ([]Finding, error)
}

// Lexi checks a text for insensitive or inconsiderate writing. It holds no global state, so
// many goroutines can share one value.
type Lexi struct {
	l *lexi.Linter
}

// NewLexi builds a lexi checker. Pass the options of an .alexrc file, or zero Options.
func NewLexi(o lexi.Options) (*Lexi, error) {
	l, err := lexi.New(o)
	if err != nil {
		return nil, err
	}
	return &Lexi{l: l}, nil
}

// Name returns the name of the checker.
func (*Lexi) Name() string { return "lexi" }

// Check scans a Markdown text.
func (x *Lexi) Check(text string) ([]Finding, error) {
	fs, err := x.l.Check(text, lexi.Markdown)
	out := make([]Finding, 0, len(fs))
	for _, f := range fs {
		out = append(out, Finding{Checker: "lexi", Line: f.Line, Column: f.Column, Rule: f.RuleID, Message: f.Message})
	}
	return out, err
}

// Run runs each checker on the text and returns all findings in order of position. A checker
// that cannot run returns an error, and Run stops there, because a text that no checker saw
// must not go out.
func Run(text string, checkers ...Checker) ([]Finding, error) {
	var all []Finding
	for _, c := range checkers {
		fs, err := c.Check(text)
		if err != nil {
			return all, fmt.Errorf("%s: %w", c.Name(), err)
		}
		all = append(all, fs...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Line != all[j].Line {
			return all[i].Line < all[j].Line
		}
		return all[i].Column < all[j].Column
	})
	return all, nil
}
