package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// healer is a forge with labels, issue edits and a pull request list. Each edit lands in log.
type healer struct {
	reader
	labels []forge.Label
	status map[int]forge.PRStatus
	log    *[]string
	open   map[int]bool
}

func (h healer) LabelList(context.Context) ([]forge.Label, error) { return h.labels, nil }
func (h healer) LabelCreate(_ context.Context, name, color, desc string) error {
	*h.log = append(*h.log, fmt.Sprintf("create %s %s %s", name, color, desc))
	return nil
}
func (h healer) LabelRename(context.Context, int64, string) error { return nil }
func (h healer) IssueEdit(_ context.Context, n int, in forge.IssueEdit) error {
	*h.log = append(*h.log, fmt.Sprintf("edit %d +%v -%v", n, in.AddLabels, in.RmLabels))
	return nil
}
func (h healer) PRStatus(_ context.Context, n int) (forge.PRStatus, error) { return h.status[n], nil }
func (h healer) PRList(_ context.Context, f forge.ListFilter) ([]forge.PRInfo, error) {
	var out []forge.PRInfo
	for _, p := range h.prs {
		if f.State == "open" && !h.open[p.Number] {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func healFixture(labels ...forge.Label) (healer, *[]string) {
	var log []string
	return healer{
		labels: labels,
		log:    &log,
		reader: reader{prs: []forge.PRInfo{
			{Number: 1, Mergeable: true},  // closed
			{Number: 2, Mergeable: true},  // open, no worker
			{Number: 3, Mergeable: true},  // worker, healthy
			{Number: 4, Mergeable: false}, // worker, conflict
			{Number: 5, Mergeable: true},  // worker, red
		}},
		open: map[int]bool{2: true, 3: true, 4: true, 5: true},
		status: map[int]forge.PRStatus{
			3: {CI: "success"}, 4: {CI: "success"}, 5: {CI: "failure"},
		},
	}, &log
}

func TestPRHeal(t *testing.T) {
	c, log := healFixture()
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "ensure"); code != ExitOK || out != "created\n" || (*log)[0] != "create ➕healing d73a4a A worker heals this pull request now" {
		t.Errorf("ensure: %d %q %v", code, out, *log)
	}
	c, log = healFixture(forge.Label{Name: "➕healing"})
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "ensure"); code != ExitOK || out != "exists\n" || len(*log) != 0 {
		t.Errorf("ensure again: %d %q %v", code, out, *log)
	}
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "add", "7"); code != ExitOK || out != "ok\n" || (*log)[0] != "edit 7 +[➕healing] -[]" {
		t.Errorf("add: %d %q %v", code, out, *log)
	}
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "remove", "3"); code != ExitOK || out != "ok\n" || (*log)[1] != "edit 3 +[] -[➕healing]" {
		t.Errorf("remove: %d %q %v", code, out, *log)
	}
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "remove", "99"); code != ExitOK || out != "absent\n" || len(*log) != 2 {
		t.Errorf("remove absent: %d %q %v", code, out, *log)
	}
}

func TestPRHealSweep(t *testing.T) {
	c, log := healFixture(forge.Label{Name: "➕healing"})
	code, out, _ := run(t, c, nil, false, "pr", "heal", "sweep", "--live", "3,4,5")
	want := "cleared\t1\tclosed\ncleared\t2\tno-worker\ncleared\t3\thealthy\n"
	if code != ExitOK || out != want || len(*log) != 3 {
		t.Errorf("sweep: %d %q %v", code, out, *log)
	}
	if code, out, _ := run(t, c, nil, false, "pr", "heal", "sweep", "--live", "none"); code != ExitOK || !strings.Contains(out, "cleared\t4\tno-worker\n") {
		t.Errorf("sweep none: %d %q", code, out)
	}
}

func TestPRHealUsage(t *testing.T) {
	c, _ := healFixture()
	for _, a := range [][]string{
		{"pr", "heal"}, {"pr", "heal", "nope"}, {"pr", "heal", "add"}, {"pr", "heal", "add", "x"},
		{"pr", "heal", "ensure", "1"}, {"pr", "heal", "sweep"}, {"pr", "heal", "sweep", "--live", "a,b"},
		{"pr", "heal", "add", "1", "--live", "1"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, false, "pr", "heal", "ensure"); code != ExitUnavailable {
		t.Errorf("no support: %d", code)
	}
}
