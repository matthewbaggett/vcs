// Package cli runs the vcs verbs. It holds no global state, so a test can drive it.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// Env is everything the verbs take from outside the process.
type Env struct {
	Stdout, Stderr io.Writer
	Getenv         func(string) string
	// IsTerminal tells whether stdin and stdout are both terminals.
	IsTerminal bool
	// RemoteURL returns the URL of the home remote.
	RemoteURL func() (string, error)
	// NewClient builds a driver for a host and a repo. It returns forge.ErrUnsupported
	// when the host has no driver.
	NewClient func(host, repo string) (forge.Client, error)
}

// IsAgent tells whether an agent called vcs.
func (e Env) IsAgent(flag string) bool {
	switch {
	case flag == "agent":
		return true
	case flag == "human":
		return false
	}
	switch e.Getenv("VCS_CALLER") {
	case "agent":
		return true
	case "human":
		return false
	}
	if e.Getenv("CLAUDECODE") == "1" || e.Getenv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return true
	}
	return !e.IsTerminal
}

type opts struct {
	pos    []string
	flags  map[string]string
	repo   string
	host   string
	caller string
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Run runs one command line and returns the exit code.
func Run(ctx context.Context, args []string, env Env) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
		return ExitUsage
	}
	if len(o.pos) < 2 {
		fmt.Fprintln(env.Stderr, "usage: vcs <noun> <verb> [args]")
		return ExitUsage
	}
	noun, name := o.pos[0], o.pos[1]
	verb, ok := Find(noun, name)
	if !ok {
		fmt.Fprintf(env.Stderr, "vcs: no verb %q %q\n", noun, name)
		return ExitUsage
	}
	o.pos = o.pos[2:]
	code := dispatch(ctx, verb, o, env)
	hint(verb, code, o, env)
	return code
}

func hint(v Verb, code int, o opts, env Env) {
	if !env.IsAgent(o.caller) {
		return
	}
	out := OutOK
	switch code {
	case ExitFail:
		out = OutFail
	case ExitUsage:
		out = OutUsage
	case ExitUnavailable:
		out = OutUnavailable
	}
	if h := v.Hints[out]; h != "" {
		fmt.Fprintf(env.Stderr, "vcs: next: %s\n", h)
	}
}

// valued flags take a value. Every other flag is an error.
var valued = map[string]bool{"repo": true, "host": true, "caller": true, "branch": true, "sha": true, "status": true, "limit": true, "tail": true}

func parse(args []string) (opts, error) {
	o := opts{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			o.pos = append(o.pos, a)
			continue
		}
		name, val, hasVal := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !valued[name] {
			return o, fmt.Errorf("unknown flag --%s", name)
		}
		if !hasVal {
			if i+1 >= len(args) {
				return o, fmt.Errorf("flag --%s needs a value", name)
			}
			i++
			val = args[i]
		}
		o.flags[name] = val
	}
	o.repo, o.host, o.caller = o.flags["repo"], o.flags["host"], o.flags["caller"]
	return o, nil
}

func usage(env Env, v Verb) int {
	fmt.Fprintf(env.Stderr, "usage: %s\n", v.Usage)
	return ExitUsage
}

func intFlag(o opts, name string, def int) (int, bool) {
	s, ok := o.flags[name]
	if !ok {
		return def, true
	}
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= 0
}

func dispatch(ctx context.Context, v Verb, o opts, env Env) int {
	if v.Noun == "docs" {
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		names, err := WritePages(o.pos[0])
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		for _, n := range names {
			fmt.Fprintln(env.Stdout, n)
		}
		return ExitOK
	}

	// Check the arguments before any call to the server.
	var n int
	var runArg string
	var limit, tail int
	var good bool
	switch v.Noun + " " + v.Name {
	case "pr status":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
			return usage(env, v)
		}
	case "ci runs":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
		if limit, good = intFlag(o, "limit", 20); !good {
			return usage(env, v)
		}
		if s := o.flags["sha"]; s != "" && !regexp.MustCompile(`^[0-9A-Fa-f]{4,64}$`).MatchString(s) {
			return usage(env, v)
		}
	case "ci jobs":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		runArg = o.pos[0]
	case "ci log":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
			return usage(env, v)
		}
		if tail, good = intFlag(o, "tail", 0); !good {
			return usage(env, v)
		}
	}

	host, repo := o.host, o.repo
	if repo == "" || host == "" {
		url, err := env.RemoteURL()
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: no home remote. Pass --repo owner/name and --host. %v\n", err)
			return ExitFail
		}
		h, r, err := forge.ParseRemote(url)
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		if host == "" {
			host = h
		}
		if repo == "" {
			repo = r
		}
	}

	// A run address names its own repo. It beats the remote.
	var runIndex int64
	if runArg != "" {
		if m := runURLRe.FindStringSubmatch(runArg); m != nil {
			host, repo = m[1], m[2]
			runIndex, _ = strconv.ParseInt(m[3], 10, 64)
		} else if id, err := strconv.ParseInt(runArg, 10, 64); err == nil && id > 0 {
			runIndex = -id
		} else {
			return usage(env, v)
		}
	}

	c, err := env.NewClient(host, repo)
	if err != nil {
		fmt.Fprintf(env.Stderr, "vcs: %s: %v\n", host, err)
		if errors.Is(err, forge.ErrUnsupported) {
			return ExitUnavailable
		}
		return ExitFail
	}
	return call(ctx, v, c, env, o, n, limit, tail, runIndex)
}

var runURLRe = regexp.MustCompile(`^https?://([^/]+)/([^/]+/[^/]+)/actions/runs/([0-9]+)(?:/.*)?$`)

func fail(env Env, err error) int {
	fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
	var fe *forge.Error
	if errors.As(err, &fe) && fe.Status == 501 {
		return ExitUnavailable
	}
	return ExitFail
}

func field(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}

func row(env Env, cols ...string) {
	for i := range cols {
		cols[i] = field(cols[i])
	}
	fmt.Fprintln(env.Stdout, strings.Join(cols, "\t"))
}

func call(ctx context.Context, v Verb, c forge.Client, env Env, o opts, n, limit, tail int, runIndex int64) int {
	switch v.Noun + " " + v.Name {
	case "pr status":
		s, err := c.PRStatus(ctx, n)
		if err != nil {
			return fail(env, err)
		}
		row(env, s.State, strconv.FormatBool(s.Draft), strconv.FormatBool(s.Mergeable),
			strings.Join(s.ConflictedFiles, ","), s.CI, s.HeadSHA, s.URL)
	case "ci runs":
		runs, err := c.CIRuns(ctx, forge.RunFilter{Branch: o.flags["branch"], SHA: o.flags["sha"], Status: o.flags["status"], Limit: limit})
		if err != nil {
			return fail(env, err)
		}
		for _, r := range runs {
			row(env, strconv.FormatInt(r.ID, 10), strconv.FormatInt(r.Index, 10), r.Status, r.Ref, r.SHA, r.Title, r.URL)
		}
	case "ci jobs":
		id := -runIndex
		if runIndex > 0 {
			// The address holds the index of the run, not its id. Find the id in the newest 250 runs.
			runs, err := c.CIRuns(ctx, forge.RunFilter{Limit: 250})
			if err != nil {
				return fail(env, err)
			}
			id = 0
			for _, r := range runs {
				if r.Index == runIndex || strings.HasSuffix(r.URL, "/actions/runs/"+strconv.FormatInt(runIndex, 10)) {
					id = r.ID
					break
				}
			}
			if id == 0 {
				return fail(env, fmt.Errorf("run number %d is not in the newest 250 runs", runIndex))
			}
		}
		jobs, err := c.CIJobs(ctx, id)
		if err != nil {
			return fail(env, err)
		}
		for _, j := range jobs {
			note := ""
			if j.NoLog {
				note = "no-log"
			}
			row(env, strconv.FormatInt(j.ID, 10), j.Name, j.Status, strconv.Itoa(j.Attempt), note)
		}
	case "ci log":
		text, err := c.CILog(ctx, int64(n))
		if errors.Is(err, forge.ErrNoLog) {
			fmt.Fprintf(env.Stderr, "vcs: job %d has no log. A wrapper job, which calls a reusable workflow, has none. Run vcs ci jobs on the run. A job marked no-log is a wrapper. Read the log of the called job\n", n)
			return ExitFail
		}
		if err != nil {
			return fail(env, err)
		}
		text = ansiRe.ReplaceAllString(text, "")
		text = strings.TrimRight(text, "\n")
		if text == "" {
			return ExitOK
		}
		lines := strings.Split(text, "\n")
		if tail > 0 && len(lines) > tail {
			lines = lines[len(lines)-tail:]
		}
		fmt.Fprintln(env.Stdout, strings.Join(lines, "\n"))
	}
	return ExitOK
}
