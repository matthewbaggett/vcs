package main

import (
	"os"
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
