package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/site"
)

var prefixRe = regexp.MustCompile(`^[0-9]+-`)

// readDocs reads the Markdown files of a directory. A name such as 10-install.md gives the slug
// install. The number sets the order.
func readDocs(dir string) ([]site.Doc, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var docs []site.Doc
	for _, n := range names {
		b, err := os.ReadFile(n)
		if err != nil {
			return nil, err
		}
		slug := prefixRe.ReplaceAllString(strings.TrimSuffix(filepath.Base(n), ".md"), "")
		docs = append(docs, site.Doc{Slug: slug, Markdown: string(b)})
	}
	return docs, nil
}

// BuildSite renders the whole site. The hand pages come from src/guide and src/reference. The
// command pages, the command list, and the exit codes page come from the command tree.
func BuildSite(src string) (map[string][]byte, error) {
	guide, err := readDocs(filepath.Join(src, "guide"))
	if err != nil {
		return nil, err
	}
	ref, err := readDocs(filepath.Join(src, "reference"))
	if err != nil {
		return nil, err
	}
	if len(guide) == 0 {
		return nil, fmt.Errorf("no guide pages in %s", filepath.Join(src, "guide"))
	}
	ref = append(ref,
		site.Doc{Slug: "commands", Markdown: commandsIndex()},
		site.Doc{Slug: "exit-codes", Markdown: exitCodesPage()},
	)
	var cmds []site.Doc
	for name, md := range Pages() {
		cmds = append(cmds, site.Doc{Slug: strings.TrimSuffix(name, ".md"), Markdown: md})
	}
	return site.Build(site.Input{Name: "vcs", Guide: guide, Reference: ref, Commands: cmds})
}

// WriteSite renders the site into dir. It removes each file of dir that the site no longer holds.
func WriteSite(src, dir string) ([]string, error) {
	files, err := BuildSite(src)
	if err != nil {
		return nil, err
	}
	var stale []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if _, ok := files[filepath.ToSlash(rel)]; !ok {
			stale = append(stale, p)
		}
		return nil
	})
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(files))
	for n, b := range files {
		p := filepath.Join(dir, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// commandsIndex lists every verb with its summary, grouped by noun.
func commandsIndex() string {
	var b strings.Builder
	b.WriteString("# Commands\n\nThe command tree generates this page and each page it links to. ")
	b.WriteString("Each verb has one page with its usage, output, flags, exit codes, and agent hints.\n")
	noun := ""
	for _, v := range Tree {
		if v.Noun != noun {
			noun = v.Noun
			fmt.Fprintf(&b, "\n## `vcs %s`\n\n| Verb | What it does |\n|---|---|\n", noun)
		}
		first := v.Summary
		if i := strings.Index(first, ". "); i > 0 {
			first = first[:i+1]
		}
		fmt.Fprintf(&b, "| [`vcs %s`](commands/%s.md) | %s |\n", v.Key(), strings.ReplaceAll(v.Key(), " ", "-"), strings.ReplaceAll(first, "|", "\\|"))
	}
	return b.String()
}

// exitCodesPage lists the exit codes, then the codes of each verb.
func exitCodesPage() string {
	var b strings.Builder
	b.WriteString("# Exit codes and agent hints\n\nThe command tree generates this page.\n\n")
	b.WriteString("## Exit codes\n\n| Code | Meaning |\n|---|---|\n")
	for _, r := range [][2]string{
		{"0", "The command did its work."},
		{"1", "The command ran and failed, or a check of a git verb failed."},
		{"2", "`stopwatch list --check` found a running stopwatch. It means found, not failed."},
		{"64", "Usage error. The arguments are wrong."},
		{"65", "The text you sent fails the prose lint. Nothing was sent."},
		{"69", "No usable driver, or the driver doesn't have the call. Stop and ask."},
		{"77", "The verb needs a web session, and there is none. Run `vcs session login`."},
	} {
		fmt.Fprintf(&b, "| %s | %s |\n", r[0], r[1])
	}
	b.WriteString("\n## Agent hints\n\nAn agent is any caller that isn't a person at a terminal. ")
	b.WriteString("The flags `--agent` and `--human`, and the variable `VCS_CALLER`, set the caller by hand. ")
	b.WriteString("An agent sees one line on stderr after each run: `vcs: next: <hint>`. ")
	b.WriteString("The hint tells the agent what to do for the outcome. A person never sees it.\n\n")
	b.WriteString("Each verb has a hint for the outcomes `ok`, `fail`, `usage`, and `unavailable`. ")
	b.WriteString("A verb that sends text also has `lint`.\n\n")
	b.WriteString("## Exit codes of each verb\n\n| Verb | Exit codes |\n|---|---|\n")
	for _, v := range Tree {
		fmt.Fprintf(&b, "| [`vcs %s`](commands/%s.md) | %s |\n", v.Key(), strings.ReplaceAll(v.Key(), " ", "-"), strings.ReplaceAll(strings.Join(v.Exits, "; "), "|", "\\|"))
	}
	return b.String()
}
