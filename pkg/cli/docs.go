package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Pages renders one Markdown page for each verb, keyed by file name.
func Pages() map[string]string {
	pages := map[string]string{}
	for _, v := range Tree {
		var b strings.Builder
		fmt.Fprintf(&b, "# The `vcs %s` command\n\n", v.Key())
		fmt.Fprintf(&b, "%s\n\n", v.Summary)
		b.WriteString("The command tree generates this page. Don't edit it by hand.\n\n")
		fmt.Fprintf(&b, "## Usage\n\n```text\n%s\n```\n\n", v.Usage)
		fmt.Fprintf(&b, "## Output\n\n%s\n\n", v.Output)
		if ex := Examples[v.Key()]; ex != "" {
			fmt.Fprintf(&b, "## Example\n\nThe output below is an example with made-up data, not a real run.\n\n```text\n%s\n```\n\n", ex)
		}
		if len(v.Flags) > 0 {
			b.WriteString("## Flags\n\n| Flag | Meaning |\n|---|---|\n")
			for _, f := range v.Flags {
				fmt.Fprintf(&b, "| `%s` | %s |\n", f.Name, f.Help)
			}
			b.WriteString("\n")
		}
		b.WriteString("## Exit codes\n\n")
		for _, e := range v.Exits {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		b.WriteString("\n## Agent hints\n\nA person never sees these. An agent sees one as a `vcs: next:` line on stderr.\n\n| Outcome | Hint |\n|---|---|\n")
		keys := make([]string, 0, len(v.Hints))
		for k := range v.Hints {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h := v.Hints[k]
			if h == "" {
				h = "none"
			}
			fmt.Fprintf(&b, "| %s | %s |\n", k, h)
		}
		pages[strings.ReplaceAll(v.Key(), " ", "-")+".md"] = b.String()
	}
	return pages
}

// WritePages writes the pages to a directory and returns the names written.
func WritePages(dir string) ([]string, error) {
	pages := Pages()
	names := make([]string, 0, len(pages))
	for n := range pages {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(pages[n]), 0o644); err != nil {
			return nil, err
		}
	}
	return names, nil
}
