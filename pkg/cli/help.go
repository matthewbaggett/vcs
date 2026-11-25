package cli

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// NounPurpose holds one line for each noun. `vcs help` shows it. A test fails when a noun has
// no line.
var NounPurpose = map[string]string{
	"attach":     "Add a file to a pull request or an issue as a comment.",
	"board":      "Read and change a project board.",
	"ci":         "Read CI runs, jobs, logs, and artifacts. Restart a run.",
	"comment":    "Post, edit, and list comments.",
	"docs":       "Build the command pages.",
	"git":        "Run git through the checked verbs.",
	"issue":      "Create, edit, list, and read issues.",
	"label":      "Manage labels.",
	"milestone":  "Manage milestones.",
	"package":    "Manage packages.",
	"pr":         "Create, read, and change pull requests. Watch one until it settles.",
	"protection": "Read and set branch protection.",
	"runners":    "Read the CI runners.",
	"secret":     "Set and list CI secrets.",
	"session":    "Sign in to the web side of the forge, for the verbs that need it.",
	"site":       "Build the command site.",
	"stopwatch":  "Start, stop, and list issue stopwatches.",
	"time":       "Log time on an issue.",
	"help":       "Show this help.",
	"version":    "Print the version.",
	"workflow":   "Read and run workflows.",
}

// nouns lists each noun of the tree once, in the order of its first verb.
func nouns() []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range Tree {
		if !seen[v.Noun] {
			seen[v.Noun] = true
			out = append(out, v.Noun)
		}
	}
	sort.Strings(out)
	return out
}

func verbsOf(noun string) []Verb {
	var out []Verb
	for _, v := range Tree {
		if v.Noun == noun {
			out = append(out, v)
		}
	}
	return out
}

// firstSentence cuts a summary at its first full stop.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

func verbLabel(v Verb) string {
	if v.Name == "" {
		return v.Noun
	}
	return v.Noun + " " + v.Name
}

// helpTop prints the nouns, each with its purpose.
func helpTop(w io.Writer) {
	fmt.Fprintln(w, "vcs: one command for Forgejo, Gitea, and GitHub.")
	fmt.Fprintln(w, "usage: vcs <noun> <verb> [args]")
	fmt.Fprintln(w)
	for _, n := range nouns() {
		fmt.Fprintf(w, "  %-11s %s\n", n, NounPurpose[n])
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "More: vcs help <noun>, and vcs help <noun> <verb>.")
}

// helpNoun prints the verbs of one noun, each with one line.
func helpNoun(w io.Writer, noun string) {
	fmt.Fprintf(w, "vcs %s: %s\n\n", noun, NounPurpose[noun])
	for _, v := range verbsOf(noun) {
		fmt.Fprintf(w, "  %-22s %s\n", verbLabel(v), firstSentence(v.Summary))
	}
	fmt.Fprintf(w, "\nMore: vcs help %s <verb>.\n", noun)
}

// helpVerb prints the page of one verb: usage, flags, exits, and one example.
func helpVerb(w io.Writer, v Verb) {
	fmt.Fprintf(w, "usage: %s\n\n%s\n", v.Usage, v.Summary)
	if v.Session {
		fmt.Fprintln(w, "\nThis verb needs a web session. Run: vcs session login")
	}
	if v.Output != "" {
		fmt.Fprintf(w, "\nOutput: %s\n", v.Output)
	}
	if len(v.Flags) > 0 {
		fmt.Fprintln(w, "\nFlags:")
		for _, f := range v.Flags {
			fmt.Fprintf(w, "  %-14s %s\n", f.Name, f.Help)
		}
	}
	if len(v.Exits) > 0 {
		fmt.Fprintln(w, "\nExit codes:")
		for _, e := range v.Exits {
			fmt.Fprintf(w, "  %s\n", e)
		}
	}
	if ex := Examples[v.Key()]; ex != "" {
		fmt.Fprintf(w, "\nExample:\n  %s\n", strings.ReplaceAll(ex, "\n", "\n  "))
	}
}

// help runs the help verb. In a terminal, a bare `vcs help` opens a menu. Anywhere else the
// output is plain text and nothing waits for input.
func help(args []string, env Env) int {
	switch len(args) {
	case 0:
		if env.IsTerminal && env.Stdin != nil {
			return helpMenu(env)
		}
		helpTop(env.Stdout)
	case 1:
		if len(verbsOf(args[0])) == 0 {
			fmt.Fprintf(env.Stderr, "vcs: no noun %q. Run: vcs help\n", args[0])
			return ExitUsage
		}
		helpNoun(env.Stdout, args[0])
	case 2, 3:
		// ci artifact get has a two-word verb.
		v, ok := Find(args[0], strings.Join(args[1:], " "))
		if !ok {
			fmt.Fprintf(env.Stderr, "vcs: no verb %q. Run: vcs help %s\n", strings.Join(args, " "), args[0])
			return ExitUsage
		}
		helpVerb(env.Stdout, v)
	default:
		fmt.Fprintln(env.Stderr, "usage: vcs help [<noun> [<verb>]]")
		return ExitUsage
	}
	return ExitOK
}

// pick shows a numbered list and reads one choice, a number or a name. It returns -1 on quit or
// at the end of the input.
func pick(in *bufio.Reader, w io.Writer, title string, items []string) int {
	fmt.Fprintf(w, "\n%s\n", title)
	for i, it := range items {
		fmt.Fprintf(w, "  %2d  %s\n", i+1, it)
	}
	for {
		fmt.Fprint(w, "Pick a number or a name, or q to quit: ")
		line, err := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" && err != nil || line == "q" {
			return -1
		}
		if n, e := strconv.Atoi(line); e == nil && n >= 1 && n <= len(items) {
			return n - 1
		}
		for i, it := range items {
			if strings.Fields(it)[0] == line {
				return i
			}
		}
		if err != nil {
			return -1
		}
		fmt.Fprintf(w, "No such choice: %s\n", line)
	}
}

// helpMenu lets the user pick a noun, then a verb, then reads the page.
func helpMenu(env Env) int {
	in := bufio.NewReader(env.Stdin)
	ns := nouns()
	items := make([]string, len(ns))
	for i, n := range ns {
		items[i] = fmt.Sprintf("%-11s %s", n, NounPurpose[n])
	}
	fmt.Fprintln(env.Stdout, "vcs: one command for Forgejo, Gitea, and GitHub.")
	i := pick(in, env.Stdout, "Nouns:", items)
	if i < 0 {
		return ExitOK
	}
	vs := verbsOf(ns[i])
	if len(vs) == 1 {
		fmt.Fprintln(env.Stdout)
		helpVerb(env.Stdout, vs[0])
		return ExitOK
	}
	items = make([]string, len(vs))
	for j, v := range vs {
		items[j] = fmt.Sprintf("%-22s %s", verbLabel(v), firstSentence(v.Summary))
	}
	j := pick(in, env.Stdout, "Verbs of "+ns[i]+":", items)
	if j < 0 {
		return ExitOK
	}
	fmt.Fprintln(env.Stdout)
	helpVerb(env.Stdout, vs[j])
	return ExitOK
}
