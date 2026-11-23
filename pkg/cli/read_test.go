package cli

import (
	"context"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// reader is a fake that also lists pull requests and issues.
type reader struct {
	fake
	prs   []forge.PRInfo
	items []forge.IssueInfo
	seen  *forge.ListFilter
}

func (r reader) PRList(_ context.Context, f forge.ListFilter) ([]forge.PRInfo, error) {
	if r.seen != nil {
		*r.seen = f
	}
	return r.prs, r.err
}
func (r reader) IssueList(_ context.Context, f forge.ListFilter) ([]forge.IssueInfo, error) {
	if r.seen != nil {
		*r.seen = f
	}
	return r.items, r.err
}
func (r reader) IssueView(_ context.Context, n int) (forge.IssueInfo, error) {
	if len(r.items) == 0 {
		return forge.IssueInfo{}, forge.ErrNotFound
	}
	return r.items[0], r.err
}

func TestPRList(t *testing.T) {
	var seen forge.ListFilter
	c := reader{
		fake:  fake{pr: forge.PRStatus{CI: "success"}},
		prs:   []forge.PRInfo{{Number: 3, Title: "A\tb", Head: "feature/x", Base: "main", Mergeable: true, Labels: []string{"a", "b"}, Milestone: "m"}},
		items: nil, seen: &seen,
	}
	code, out, _ := run(t, c, nil, true, "pr", "list", "--state", "all", "--label", "a, b", "--milestone", "m", "--limit", "5")
	if code != 0 || out != "3\tA b\tfeature/x\tmain\ttrue\t\ta,b\tm\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if seen.State != "all" || len(seen.Labels) != 2 || seen.Labels[1] != "b" || seen.Milestone != "m" || seen.Limit != 5 {
		t.Errorf("filter = %+v", seen)
	}
	// --ci adds one status call for each row.
	if _, out, _ := run(t, c, nil, true, "pr", "list", "--ci"); out != "3\tA b\tfeature/x\tmain\ttrue\tsuccess\ta,b\tm\n" {
		t.Errorf("with ci: %q", out)
	}
	// An empty list is not an error.
	if code, out, _ := run(t, reader{}, nil, true, "pr", "list"); code != 0 || out != "" {
		t.Errorf("empty: %d %q", code, out)
	}
}

func TestIssueListAndView(t *testing.T) {
	c := reader{items: []forge.IssueInfo{{
		Number: 8, Title: "Thin scripts", State: "open", Labels: []string{"plan"}, Milestone: "v1",
		Body: "The body.\n", Comments: []forge.Comment{{Author: "ann", Created: "2026-01-02", Body: "Hi\n"}},
	}}}
	if _, out, _ := run(t, c, nil, true, "issue", "list"); out != "8\tThin scripts\topen\tplan\tv1\n" {
		t.Errorf("list: %q", out)
	}
	_, out, _ := run(t, c, nil, true, "issue", "view", "8")
	want := "8\tThin scripts\topen\tplan\tv1\n\nThe body.\n\ncomment\tann\t2026-01-02\nHi\n"
	if out != want {
		t.Errorf("view: %q", out)
	}
	if code, _, _ := run(t, reader{}, nil, true, "issue", "view", "8"); code != ExitFail {
		t.Errorf("missing issue: exit %d", code)
	}
}

func TestReadVerbsUsageAndUnavailable(t *testing.T) {
	for _, args := range [][]string{
		{"pr", "list", "extra"}, {"pr", "list", "--state", "bogus"}, {"pr", "list", "--limit", "x"},
		{"issue", "list", "--ci"}, {"issue", "view"}, {"issue", "view", "x"},
		{"ci", "failed"}, {"ci", "failed", "zz"}, {"ci", "failed", "12345678901234567890123456789012345678901"},
	} {
		if code, _, _ := run(t, reader{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v exit %d, want 64", args, code)
		}
	}
	// A driver without the Reader interface exits 69.
	bare := struct{ forge.Client }{fake{}}
	for _, args := range [][]string{{"pr", "list"}, {"issue", "list"}, {"issue", "view", "1"}} {
		if code, _, errs := run(t, bare, nil, true, args...); code != ExitUnavailable || !strings.Contains(errs, "no ") {
			t.Errorf("%v: %d %q", args, code, errs)
		}
	}
}

func TestCIFailed(t *testing.T) {
	c := fake{
		pr: forge.PRStatus{HeadSHA: "abc1234"},
		runs: []forge.Run{
			{ID: 9, Status: "failure", SHA: "abc1234"},
			{ID: 8, Status: "success", SHA: "abc1234"},
		},
		jobs: []forge.Job{
			{ID: 7, Name: "gate", Status: "failure"},
			{ID: 6, Name: "wrapper", Status: "failure", NoLog: true},
			{ID: 5, Name: "ok", Status: "success"},
		},
		log: "start\n2026-01-01T00:00:00Z ::error file=a.go::boom\tgo\nend\n",
	}
	code, out, _ := run(t, c, nil, true, "ci", "failed", "12")
	if code != 0 || out != "9\t7\tgate\tboom go\n9\t6\twrapper\t\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	// A sha works without a status call.
	if _, out2, _ := run(t, c, nil, true, "ci", "failed", "abc1234"); out2 != out {
		t.Errorf("by sha: %q", out2)
	}
	// A log with no ::error:: line gives an empty column. A plain ::error:: has no attributes.
	c.log = "::error::plain text\n"
	if _, out, _ := run(t, c, nil, true, "ci", "failed", "12"); !strings.HasPrefix(out, "9\t7\tgate\tplain text\n") {
		t.Errorf("plain: %q", out)
	}
	c.log = "nothing here\n"
	if _, out, _ := run(t, c, nil, true, "ci", "failed", "12"); !strings.HasPrefix(out, "9\t7\tgate\t\n") {
		t.Errorf("no error line: %q", out)
	}
	// A run of another commit is an error and prints no row.
	c.runs = []forge.Run{{ID: 9, Status: "failure", SHA: "ffff999"}}
	if code, out, errs := run(t, c, nil, true, "ci", "failed", "12"); code != ExitFail || out != "" || !strings.Contains(errs, "belongs to commit") {
		t.Errorf("wrong commit: %d %q %q", code, out, errs)
	}
}
