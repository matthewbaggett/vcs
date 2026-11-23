package lint

import (
	"github.com/vale-cli/vale/v3/pkg/vale"
)

// Vale checks a text with the Vale facade, in process.
type Vale struct {
	c *vale.Client
}

// NewVale builds a Vale checker. An empty ConfigPath makes Vale search for a .vale.ini.
func NewVale(o vale.Options) (*Vale, error) {
	c, err := vale.New(o)
	if err != nil {
		return nil, err
	}
	return &Vale{c: c}, nil
}

// Name returns the name of the checker.
func (*Vale) Name() string { return "vale" }

// Check scans a Markdown text. Each alert is a finding, whatever its severity.
func (v *Vale) Check(text string) ([]Finding, error) {
	r, err := v.c.LintString(text, ".md")
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, alerts := range r.Files {
		for _, a := range alerts {
			col := 0
			if len(a.Span) > 0 {
				col = a.Span[0]
			}
			out = append(out, Finding{Checker: "vale", Line: a.Line, Column: col, Rule: a.Check, Message: a.Message})
		}
	}
	return out, nil
}
