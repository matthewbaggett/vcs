package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vale-cli/vale/v3/pkg/vale"
)

func TestValeFindsAlert(t *testing.T) {
	dir := t.TempDir()
	styles := filepath.Join(dir, "styles", "T")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	rule := "extends: existence\nmessage: \"Avoid '%s'.\"\nlevel: error\nignorecase: true\ntokens:\n  - utilize\n"
	if err := os.WriteFile(filepath.Join(styles, "Words.yml"), []byte(rule), 0o600); err != nil {
		t.Fatal(err)
	}
	ini := "StylesPath = styles\nMinAlertLevel = suggestion\n[*.md]\nBasedOnStyles = T\n"
	cfg := filepath.Join(dir, ".vale.ini")
	if err := os.WriteFile(cfg, []byte(ini), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := NewVale(vale.Options{ConfigPath: cfg})
	if err != nil {
		t.Fatal(err)
	}
	fs, err := Run("We utilize it.\n", v)
	if err != nil || len(fs) != 1 || fs[0].Checker != "vale" || fs[0].Line != 1 {
		t.Fatalf("fs=%v err=%v", fs, err)
	}
	fs, err = Run("We use it.\n", v)
	if err != nil || len(fs) != 0 {
		t.Fatalf("fs=%v err=%v", fs, err)
	}
}
