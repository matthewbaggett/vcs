package guard

import "testing"

func TestBranch(t *testing.T) {
	for _, ok := range []string{"feature/pr-watch", "fix/x", "ci/quality-gate-workflow", "chore/wire-vale"} {
		if r := Branch(ok); r != nil {
			t.Errorf("%s refused: %v", ok, r)
		}
	}
	for _, bad := range []string{"main", "feat/pr-watch", "fix/Upper", "fix/", "fix/a_b", "wip"} {
		if Branch(bad) == nil {
			t.Errorf("%s allowed", bad)
		}
	}
}

func TestCommitOnMain(t *testing.T) {
	if Commit("main", "Add a thing") == nil {
		t.Error("main allowed")
	}
	if r := Commit("fix/x", "Add a thing"); r != nil {
		t.Errorf("refused: %v", r)
	}
}

func TestCredit(t *testing.T) {
	for _, bad := range []string{
		"Add x\n\nCo-Authored-By: Claude Sonnet <noreply@anthropic.com>",
		"Body\n\n🤖 Generated with [Claude Code](https://claude.com/claude-code)",
		"co-authored-by: Copilot <x@y>",
	} {
		if Credit(bad) == nil {
			t.Errorf("allowed %q", bad)
		}
	}
	if r := Credit("Add x\n\nCo-authored-by: Ada Lovelace <ada@example.com>"); r != nil {
		t.Errorf("refused a person: %v", r)
	}
}

func TestPush(t *testing.T) {
	for _, bad := range [][]string{{"--force"}, {"-f"}, {"--force-with-lease"}, {"origin", "+main"}, {"--no-verify"}} {
		if Push(bad) == nil {
			t.Errorf("allowed %v", bad)
		}
	}
	if r := Push([]string{"-u", "origin", "fix/x"}); r != nil {
		t.Errorf("refused: %v", r)
	}
}
