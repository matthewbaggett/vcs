package site

import (
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	files, err := Build(Input{
		Name:      "x",
		Guide:     []Doc{{"home", "# Home\n\n[Next](two.md)\n"}},
		Reference: []Doc{{"two", "# Two\n"}},
		Commands:  []Doc{{"pr-tick", "# The `vcs pr tick` command\n"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files["index.html"]), `href="two.html"`) {
		t.Error("link not rewritten")
	}
	c := string(files["commands/pr-tick.html"])
	if !strings.Contains(c, `href="../assets/style.css"`) || !strings.Contains(c, `aria-current="page"`) {
		t.Error("command page paths or nav wrong")
	}
	if _, ok := files[".nojekyll"]; !ok {
		t.Error("no .nojekyll")
	}
	if _, err := Build(Input{Guide: []Doc{{"a", "no heading"}}}); err == nil {
		t.Error("want an error for a page with no heading")
	}
	if Title("# The `vcs x` command") != "The vcs x command" {
		t.Error("title")
	}
}
