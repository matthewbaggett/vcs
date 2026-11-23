// Package cli runs the vcs verbs. It holds no global state, so a test can drive it.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/guard"
	"git.grey.ooo/packages/vcs/pkg/lint"
	"git.grey.ooo/packages/vcs/pkg/session"
)

// Env is everything the verbs take from outside the process.
type Env struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	// ConfigDir holds the web sessions.
	ConfigDir string
	// BaseURL returns the web address of a host, such as https://git.grey.ooo.
	BaseURL func(host string) string
	Getenv  func(string) string
	// IsTerminal tells whether stdin and stdout are both terminals.
	IsTerminal bool
	// RemoteURL returns the URL of the home remote.
	RemoteURL func() (string, error)
	// NewClient builds a driver for a host and a repo. It returns forge.ErrUnsupported
	// when the host has no driver.
	NewClient func(host, repo string) (forge.Client, error)
	// Sleep waits between two polls. A nil value waits in real time.
	Sleep func(ctx context.Context, d time.Duration)
	// Checkers lint each text that a write verb sends. A finding stops the send.
	Checkers []lint.Checker
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
	if len(o.pos) < 2 && !(len(o.pos) == 1 && o.pos[0] == "attach") {
		fmt.Fprintln(env.Stderr, "usage: vcs <noun> <verb> [args]")
		return ExitUsage
	}
	noun, name, rest := o.pos[0], "", o.pos[1:]
	if noun != "attach" { // attach is a noun with no verb
		name, rest = o.pos[1], o.pos[2:]
	}
	verb, ok := Find(noun, name)
	if !ok {
		fmt.Fprintf(env.Stderr, "vcs: no verb %q %q\n", noun, name)
		return ExitUsage
	}
	o.pos = rest
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
	case ExitLint:
		out = OutLint
	case ExitUnavailable:
		out = OutUnavailable
	case ExitNoSession:
		out = OutNoSession
	}
	if h := v.Hints[out]; h != "" {
		fmt.Fprintf(env.Stderr, "vcs: next: %s\n", h)
	}
}

// valued flags take a value. Every other flag is an error.
var valued = map[string]bool{"user": true, "title": true, "repo": true, "host": true, "caller": true, "branch": true, "sha": true, "status": true, "limit": true, "tail": true, "interval": true, "timeout": true, "body": true, "body-file": true, "head": true, "base": true}

// boolean flags take no value.
var boolean = map[string]bool{"failed": true, "comment": true, "draft": true}

func parse(args []string) (opts, error) {
	o := opts{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			o.pos = append(o.pos, a)
			continue
		}
		name, val, hasVal := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if boolean[name] {
			o.flags[name] = "true"
			continue
		}
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

	if v.Noun == "session" {
		return sessionVerb(ctx, v, o, env)
	}

	// Check the arguments before any call to the server.
	var n int
	var runArg string
	var limit, tail int
	var good bool
	switch v.Key() {
	case "pr status", "pr watch":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
			return usage(env, v)
		}
		if v.Name == "watch" {
			if i, ok := intFlag(o, "interval", 30); !ok || i < 1 {
				return usage(env, v)
			}
			if t, ok := intFlag(o, "timeout", 3600); !ok || t < 1 {
				return usage(env, v)
			}
		}
	case "pr create":
		if len(o.pos) != 1 || o.flags["head"] == "" || (o.flags["body"] == "") == (o.flags["body-file"] == "") {
			return usage(env, v)
		}
		if f := o.flags["body-file"]; f != "" {
			raw, err := readText(env, f)
			if err != nil {
				fmt.Fprintf(env.Stderr, "vcs: cannot read a body from %s\n", f)
				return ExitUsage
			}
			o.flags["body"] = raw
		}
		if o.flags["base"] == "" {
			o.flags["base"] = "main"
		}
		var refuse []string
		refuse = append(refuse, guard.Branch(o.flags["head"])...)
		refuse = append(refuse, guard.Credit(o.pos[0]+"\n"+o.flags["body"])...)
		if len(refuse) > 0 {
			for _, r := range refuse {
				fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r)
			}
			return ExitLint
		}
		if code := lintText(env, o.pos[0]); code != ExitOK {
			return code
		}
		if code := lintText(env, o.flags["body"]); code != ExitOK {
			return code
		}
	case "comment add":
		if len(o.pos) != 1 || (o.flags["body"] == "") == (o.flags["body-file"] == "") {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
			return usage(env, v)
		}
		if f := o.flags["body-file"]; f != "" {
			var raw []byte
			if f == "-" {
				raw, err = io.ReadAll(env.Stdin)
			} else {
				raw, err = os.ReadFile(f)
			}
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				fmt.Fprintf(env.Stderr, "vcs: cannot read a comment text from %s\n", f)
				return ExitUsage
			}
			o.flags["body"] = string(raw)
		}
		if code := lintText(env, o.flags["body"]); code != ExitOK {
			return code
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
	case "ci jobs", "ci rerun":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		runArg = o.pos[0]
		if v.Name == "jobs" && o.flags["failed"] != "" {
			return usage(env, v)
		}
	case "attach":
		if len(o.pos) < 2 {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
			return usage(env, v)
		}
		for _, f := range o.pos[1:] {
			st, err := os.Lstat(f)
			if err != nil || !st.Mode().IsRegular() {
				fmt.Fprintf(env.Stderr, "vcs: %s is not a plain file\n", f)
				return ExitUsage
			}
		}
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

// resolveRun turns the run argument into a run id. A negative value is an id already.
// A positive value is the index from a run address, and the id is in the newest 250 runs.
func resolveRun(ctx context.Context, c forge.Client, runIndex int64) (int64, error) {
	if runIndex < 0 {
		return -runIndex, nil
	}
	runs, err := c.CIRuns(ctx, forge.RunFilter{Limit: 250})
	if err != nil {
		return 0, err
	}
	for _, r := range runs {
		if r.Index == runIndex || strings.HasSuffix(r.URL, "/actions/runs/"+strconv.FormatInt(runIndex, 10)) {
			return r.ID, nil
		}
	}
	return 0, fmt.Errorf("run number %d is not in the newest 250 runs", runIndex)
}

// lintText runs the checkers on a text that a verb is about to send. A finding gives exit 65,
// and so does a checker that cannot run, because a text that no checker saw must not go out.
func lintText(env Env, text string) int {
	if len(env.Checkers) == 0 {
		return ExitOK
	}
	fs, err := lint.Run(text, env.Checkers...)
	for _, f := range fs {
		fmt.Fprintf(env.Stderr, "vcs: lint: %s\n", f)
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "vcs: lint: %v\n", err)
		return ExitLint
	}
	if len(fs) > 0 {
		fmt.Fprintf(env.Stderr, "vcs: lint: %d finding(s). Nothing was sent\n", len(fs))
		return ExitLint
	}
	return ExitOK
}

func fail(env Env, err error) int {
	fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
	if errors.Is(err, forge.ErrNoSession) {
		return ExitNoSession
	}
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
	switch v.Key() {
	case "pr watch":
		return watch(ctx, c, env, n, o)
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
		id, err := resolveRun(ctx, c, runIndex)
		if err != nil {
			return fail(env, err)
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
	case "ci rerun":
		id, err := resolveRun(ctx, c, runIndex)
		if err != nil {
			return fail(env, err)
		}
		if err := c.CIRerun(ctx, id, o.flags["failed"] == "true"); err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "rerun\t%d\n", id)
	case "attach":
		var text string
		if o.flags["comment"] == "true" {
			title := o.flags["title"]
			if title == "" {
				title = "Screenshots"
			}
			text = title
			if code := lintText(env, text); code != ExitOK {
				return code
			}
		}
		assets, err := c.Attach(ctx, n, o.pos[1:])
		for _, a := range assets {
			row(env, a.Name, a.URL)
		}
		if err != nil {
			return fail(env, err)
		}
		if o.flags["comment"] == "true" {
			var b strings.Builder
			b.WriteString(text + "\n")
			for _, a := range assets {
				fmt.Fprintf(&b, "\n![%s](%s)", a.Name, a.URL)
			}
			if err := c.Comment(ctx, n, b.String()); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "comment posted")
		}
	case "pr create":
		num, addr, err := c.CreatePR(ctx, forge.NewPR{Title: o.pos[0], Body: o.flags["body"], Head: o.flags["head"], Base: o.flags["base"], Draft: o.flags["draft"] == "true"})
		if err != nil {
			return fail(env, err)
		}
		row(env, strconv.Itoa(num), addr)
	case "comment add":
		if err := c.Comment(ctx, n, o.flags["body"]); err != nil {
			return fail(env, err)
		}
		fmt.Fprintln(env.Stdout, "comment posted")
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

func baseURL(env Env, host string) string {
	if env.BaseURL != nil {
		return env.BaseURL(host)
	}
	return "https://" + host
}

// sessionVerb runs the verbs of the session noun. They need no driver.
func sessionVerb(ctx context.Context, v Verb, o opts, env Env) int {
	if (v.Name != "login" && len(o.pos) != 0) || (v.Name == "import" && len(o.pos) != 0) {
		return usage(env, v)
	}
	host := o.host
	if host == "" {
		url, err := env.RemoteURL()
		if err == nil {
			host, _, err = forge.ParseRemote(url)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: no home remote. Pass --host. %v\n", err)
			return ExitFail
		}
	}
	base := baseURL(env, host)
	switch v.Name {
	case "login":
		user := o.flags["user"]
		if user == "" {
			user = env.Getenv("VCS_USER")
		}
		pass := env.Getenv("VCS_PASSWORD")
		if user == "" || pass == "" || len(o.pos) != 0 {
			fmt.Fprintln(env.Stderr, "vcs: give --user, and set VCS_PASSWORD")
			return usage(env, v)
		}
		val, err := session.Login(ctx, base, user, pass)
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		if err := session.Save(env.ConfigDir, host, val); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
	case "import":
		data, err := io.ReadAll(io.LimitReader(env.Stdin, 4096))
		val := strings.Join(strings.Fields(string(data)), "")
		if err != nil || val == "" {
			return usage(env, v)
		}
		if err := session.Check(ctx, base, val); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		if err := session.Save(env.ConfigDir, host, val); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
	case "status":
		val, err := session.Load(env.ConfigDir, host)
		if err == nil {
			err = session.Check(ctx, base, val)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: no good session for %s. %v\n", host, err)
			return ExitNoSession
		}
	case "logout":
		if err := session.Remove(env.ConfigDir, host); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
	}
	fmt.Fprintln(env.Stdout, "ok")
	return ExitOK
}

// readText reads a body from a file, or from standard input for "-". An empty text is an error.
func readText(env Env, name string) (string, error) {
	var raw []byte
	var err error
	if name == "-" {
		raw, err = io.ReadAll(env.Stdin)
	} else {
		raw, err = os.ReadFile(name)
	}
	if err == nil && strings.TrimSpace(string(raw)) == "" {
		err = errors.New("empty")
	}
	return string(raw), err
}
