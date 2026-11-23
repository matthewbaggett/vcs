package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The prose lint needs .vale.ini at the root. It links into the agent-standards checkout.
func TestValeConfigResolves(t *testing.T) {
	if _, err := os.Stat("../../.claude/agent-standards/tools/vale/vale.ini"); err != nil {
		t.Skip("the agent-standards checkout is empty")
	}
	b, err := os.ReadFile("../../.vale.ini")
	if err != nil {
		t.Fatalf(".vale.ini does not resolve: %v", err)
	}
	if !strings.Contains(string(b), "StylesPath") {
		t.Error(".vale.ini has no StylesPath")
	}
}

// The token of a tea login never reaches the output of the binary, even when the call fails.
func TestTeaTokenNeverPrinted(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "vcs")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	secret := "made-up-secret-token-0123456789"
	cfg := filepath.Join(dir, "config.yml")
	body := "logins:\n  - name: t\n    url: https://127.0.0.1:1\n    token: " + secret + "\n    default: true\n"
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"pr", "status", "1", "--host", "127.0.0.1:1", "--repo", "a/b"}, {"pr", "list", "--host", "127.0.0.1:1", "--repo", "a/b"}} {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "TEA_CONFIG="+cfg, "VCS_TOKEN=", "HOME="+dir)
		out, _ := cmd.CombinedOutput()
		if strings.Contains(string(out), secret) {
			t.Errorf("vcs %v printed the token", args)
		}
	}
}
