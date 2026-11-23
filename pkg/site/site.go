// Package site turns Markdown pages into a static HTML site. The output uses relative links,
// so a host can serve it from any path, such as GitHub Pages under /vcs/. It needs no script.
package site

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// Doc is one Markdown page. Slug names it: "install" gives install.html. A Doc of the Commands
// list lands in the commands/ directory. Title comes from the first heading.
type Doc struct {
	Slug     string
	Markdown string
}

// Input holds every page of the site, in nav order.
type Input struct {
	// Name is the title of the site.
	Name string
	// Guide pages show first in the nav, in the order given. The first one is the home page.
	Guide []Doc
	// Reference pages show after the guide.
	Reference []Doc
	// Commands are one page for each verb. They live in commands/.
	Commands []Doc
}

var (
	titleRe = regexp.MustCompile(`(?m)^# (.+)$`)
	hrefRe  = regexp.MustCompile(`href="([^"#:]+)\.md(#[^"]*)?"`)
	tickRe  = regexp.MustCompile("`([^`]*)`")
)

// Title returns the first heading of a page, without the code marks.
func Title(md string) string {
	m := titleRe.FindStringSubmatch(md)
	if m == nil {
		return ""
	}
	return tickRe.ReplaceAllString(m[1], "$1")
}

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.Table),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
)

// Build renders the site. It returns each file as a slash path and its bytes.
func Build(in Input) (map[string][]byte, error) {
	files := map[string][]byte{
		"assets/style.css": []byte(stylesheet),
		".nojekyll":        {},
	}
	type entry struct {
		path, title, group string
		md                 string
	}
	var all []entry
	for i, d := range in.Guide {
		p := d.Slug + ".html"
		if i == 0 {
			p = "index.html"
		}
		all = append(all, entry{p, Title(d.Markdown), "Guide", d.Markdown})
	}
	for _, d := range in.Reference {
		all = append(all, entry{d.Slug + ".html", Title(d.Markdown), "Reference", d.Markdown})
	}
	cmds := append([]Doc(nil), in.Commands...)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Slug < cmds[j].Slug })
	for _, d := range cmds {
		noun := strings.SplitN(d.Slug, "-", 2)[0]
		all = append(all, entry{"commands/" + d.Slug + ".html", Title(d.Markdown), noun, d.Markdown})
	}

	// The nav lists the commands by noun. It is the same on each page, with relative links.
	for _, e := range all {
		if e.title == "" {
			return nil, fmt.Errorf("site: page %s has no heading", e.path)
		}
	}
	for _, e := range all {
		depth := strings.Count(e.path, "/")
		root := strings.Repeat("../", depth)
		var body bytes.Buffer
		if err := markdown.Convert([]byte(e.md), &body); err != nil {
			return nil, fmt.Errorf("site: %s: %w", e.path, err)
		}
		content := body.String()
		if depth == 0 {
			content = hrefRe.ReplaceAllString(content, `href="$1.html$2"`)
		}
		var nav strings.Builder
		group := ""
		for _, n := range all {
			if n.group != group {
				if group != "" {
					nav.WriteString("</ul>\n")
				}
				group = n.group
				label := group
				if n.group != "Guide" && n.group != "Reference" {
					label = "vcs " + group
				}
				fmt.Fprintf(&nav, "<h2>%s</h2>\n<ul>\n", html.EscapeString(label))
			}
			cur := ""
			if n.path == e.path {
				cur = ` aria-current="page"`
			}
			fmt.Fprintf(&nav, "<li><a href=\"%s%s\"%s>%s</a></li>\n", root, n.path, cur, html.EscapeString(shortTitle(n.title)))
		}
		nav.WriteString("</ul>\n")
		var page bytes.Buffer
		err := pageTemplate.Execute(&page, map[string]any{
			"Site": in.Name, "Title": e.title, "Root": root, "Nav": template.HTML(nav.String()), "Body": template.HTML(content),
		})
		if err != nil {
			return nil, err
		}
		files[e.path] = page.Bytes()
	}
	return files, nil
}

// shortTitle shortens the title of a command page for the nav: "The vcs pr tick command" is "pr tick".
func shortTitle(t string) string {
	t = strings.TrimPrefix(t, "The vcs ")
	return strings.TrimSuffix(t, " command")
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} | {{.Site}}</title>
<link rel="stylesheet" href="{{.Root}}assets/style.css">
</head>
<body>
<header class="top"><a class="brand" href="{{.Root}}index.html">{{.Site}}</a><a class="jump" href="#pages">Pages</a></header>
<div class="layout">
<main>
{{.Body}}
</main>
<nav class="side" id="pages" aria-label="Pages">
{{.Nav}}</nav>
</div>
</body>
</html>
`))
