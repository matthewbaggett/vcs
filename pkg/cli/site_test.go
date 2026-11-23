package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The committed site must match the command tree and the hand pages.
// Run `go run ./cmd/vcs site gen docs/Site site` to fix a drift.
func TestCommittedSiteMatchesTree(t *testing.T) {
	root := filepath.Join("..", "..")
	want, err := BuildSite(filepath.Join(root, "docs", "Site"))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "site")
	for name, b := range want {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil || string(got) != string(b) {
			t.Errorf("site/%s is out of date. Run: go run ./cmd/vcs site gen docs/Site site", name)
		}
	}
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if _, ok := want[filepath.ToSlash(rel)]; !ok {
			t.Errorf("site/%s has no source", rel)
		}
		return nil
	})
}

func TestSiteHasPageForEachVerb(t *testing.T) {
	files, err := BuildSite(filepath.Join("..", "..", "docs", "Site"))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range Tree {
		p := "commands/" + strings.ReplaceAll(v.Key(), " ", "-") + ".html"
		if _, ok := files[p]; !ok {
			t.Errorf("no page %s", p)
		}
	}
	for n, b := range files {
		if strings.HasSuffix(n, ".html") && strings.Contains(string(b), `href="/`) {
			t.Errorf("%s has an absolute link", n)
		}
	}
}

// The site never names the Forgejo host.
func TestSiteNamesNoForgejoHost(t *testing.T) {
	files, err := BuildSite(filepath.Join("..", "..", "docs", "Site"))
	if err != nil {
		t.Fatal(err)
	}
	for n, b := range files {
		if strings.Contains(string(b), "git.grey.ooo") {
			t.Errorf("%s names the Forgejo host", n)
		}
	}
}

func TestSiteGenUsage(t *testing.T) {
	if code, _, _ := run(t, fake{}, nil, true, "site", "gen", "only-one"); code != ExitUsage {
		t.Errorf("exit %d, want 64", code)
	}
	dir := t.TempDir()
	if code, _, _ := run(t, fake{}, nil, true, "site", "gen", filepath.Join("..", "..", "docs", "Site"), dir); code != ExitOK {
		t.Errorf("gen exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Error(err)
	}
	if code, _, _ := run(t, fake{}, nil, true, "site", "gen", t.TempDir(), dir); code != ExitFail {
		t.Errorf("empty source exit %d, want 1", code)
	}
}
