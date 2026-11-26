// Package cli runs the vcs verbs. It holds no global state, so a test can drive it.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	// ReadPassword reads a password from the terminal without an echo. A nil value turns the
	// login prompt off.
	ReadPassword func() (string, error)
	// KnownUser returns the user name that an existing tea or fj login holds for a host, or "".
	KnownUser func(host string) string
	// RemoteURL returns the URL of the home remote.
	RemoteURL func() (string, error)
	// NewClient builds a driver for a host and a repo. It returns forge.ErrUnsupported
	// when the host has no driver.
	NewClient func(host, repo string) (forge.Client, error)
	// Sleep waits between two polls. A nil value waits in real time.
	Sleep func(ctx context.Context, d time.Duration)
	// Now and Hostname feed the trace stamp. A nil value reads the clock and the host.
	Now      func() time.Time
	Hostname func() string
	// Checkers lint each text that a write verb sends. A finding stops the send.
	Checkers []lint.Checker
	// Exec runs a program with the given streams and returns its exit code. The git verbs use it.
	Exec func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int
	// Version is the release tag, or dev-<sha> for a dev build. `vcs version` prints it.
	Version string
	// Pid and PidAlive feed the pid file of pr watch. A nil value reads the real process table.
	Pid      func() int
	PidAlive func(pid int) bool
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
	if len(args) >= 2 && args[0] == "git" {
		// The git verbs pass their arguments to git. The flag parser would refuse most of them.
		// A leading -C <dir> works as it does for git, and it comes before the verb.
		rest, dir := args[1:], []string(nil)
		if rest[0] == "-C" {
			if len(rest) < 3 || rest[1] == "" {
				fmt.Fprintln(env.Stderr, "usage: vcs git -C <dir> <verb> [args]")
				return ExitUsage
			}
			dir, rest = rest[:2], rest[2:]
		}
		verb, ok := Find("git", rest[0])
		if !ok {
			fmt.Fprintf(env.Stderr, "vcs: no verb %q %q\n", "git", rest[0])
			return ExitUsage
		}
		code := gitVerb(ctx, verb, append(dir, rest[1:]...), env)
		hint(verb, code, opts{}, env)
		return code
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		verb, _ := Find("version", "")
		v := env.Version
		if v == "" {
			v = "dev"
		}
		fmt.Fprintln(env.Stdout, v)
		hint(verb, ExitOK, opts{}, env)
		return ExitOK
	}
	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		return help(args[1:], env)
	}
	o, err := parse(args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
		return ExitUsage
	}
	if len(o.pos) < 2 && !(len(o.pos) == 1 && o.pos[0] == "attach") {
		fmt.Fprintln(env.Stderr, "usage: vcs <noun> <verb> [args]. Run: vcs help")
		return ExitUsage
	}
	noun, name, rest := o.pos[0], "", o.pos[1:]
	if noun != "attach" { // attach is a noun with no verb
		name, rest = o.pos[1], o.pos[2:]
	}
	if noun == "ci" && name == "artifact" && len(rest) > 0 && rest[0] == "get" {
		name, rest = "artifact get", rest[1:]
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
	case ExitClosed:
		if v.Hints[OutClosed] != "" {
			out = OutClosed
		}
	}
	if h := v.Hints[out]; h != "" {
		fmt.Fprintf(env.Stderr, "vcs: next: %s\n", h)
	}
}

// valued flags take a value. Every other flag is an error.
var valued = map[string]bool{"user": true, "title": true, "repo": true, "host": true, "caller": true, "branch": true, "sha": true, "status": true, "limit": true, "tail": true, "o": true, "interval": true, "timeout": true, "body": true, "body-file": true, "head": true, "base": true, "state": true, "label": true, "milestone": true, "method": true, "due": true, "desc": true, "add-label": true, "rm-label": true, "live": true, "query": true, "trace": true, "remote": true, "kind": true, "value-file": true, "from-env": true, "ref": true, "stale-days": true, "max-mib": true, "sha256": true}

// boolean flags take no value.
var boolean = map[string]bool{"failed": true, "comment": true, "all": true, "draft": true, "ci": true, "delete-branch": true, "mine": true, "check": true, "dry-run": true, "pushed": true, "latest": true, "description": true, "allow-newest": true}

func parse(args []string) (opts, error) {
	o := opts{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-o" {
			a = "--o"
		}
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
		// pr watch takes --status with no value. ci runs takes --status <state>.
		if !hasVal && name == "status" && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "--")) {
			o.flags[name] = "true"
			continue
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

	if v.Noun == "site" {
		if len(o.pos) != 2 {
			return usage(env, v)
		}
		names, err := WriteSite(o.pos[0], o.pos[1])
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
	case "pr status", "pr watch", "pr ready", "pr draft", "pr close", "pr rebase":
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
	case "issue create":
		if len(o.pos) != 1 || (o.flags["body"] == "") == (o.flags["body-file"] == "") {
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
		if r := guard.Credit(o.pos[0] + "\n" + o.flags["body"]); len(r) > 0 {
			fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r[0])
			return ExitLint
		}
		if code := lintText(env, o.pos[0]); code != ExitOK {
			return code
		}
		if code := lintText(env, o.flags["body"]); code != ExitOK {
			return code
		}
		o.flags["body"] = stampTrace(env, o.flags, o.flags["body"])
	case "pr list", "issue list":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
		if s := o.flags["state"]; s != "" && s != "open" && s != "closed" && s != "all" {
			return usage(env, v)
		}
		if limit, good = intFlag(o, "limit", 0); !good {
			return usage(env, v)
		}
		if v.Noun == "issue" && o.flags["ci"] != "" {
			return usage(env, v)
		}
	case "issue newer":
		if len(o.pos) != 1 {
			return usage(env, v)
		}
		if m, err := strconv.Atoi(o.pos[0]); err != nil || m < 0 {
			return usage(env, v)
		}
		if s := o.flags["state"]; s != "" && s != "open" && s != "all" {
			return usage(env, v)
		}
		if k := o.flags["kind"]; k != "" && k != "issue" && k != "any" {
			return usage(env, v)
		}
	case "issue top":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	case "issue view", "issue timeline":
		if len(o.pos) != 1 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
		n, _ = strconv.Atoi(o.pos[0])
	case "pr heal":
		if code := healCheck(env, v, o); code >= 0 {
			return code
		}
	case "pr tick":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	case "pr merge":
		switch o.flags["method"] {
		case "", "merge", "rebase", "rebase-merge", "squash":
		default:
			return usage(env, v)
		}
		if len(o.pos) != 1 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
		n, _ = strconv.Atoi(o.pos[0])
	case "pr statuses":
		if len(o.pos) != 1 || !regexp.MustCompile(`^[0-9A-Za-z._/-]+$`).MatchString(o.pos[0]) {
			return usage(env, v)
		}
	case "ci failed":
		if len(o.pos) != 1 || !regexp.MustCompile(`^([0-9]{1,6}|[0-9A-Fa-f]{7,40})$`).MatchString(o.pos[0]) {
			return usage(env, v)
		}
	case "milestone list", "milestone view", "milestone create":
		if code := milestoneCheck(env, v, o); code >= 0 {
			return code
		}
	case "time add", "time list", "time total", "stopwatch list":
		if code := timeCheck(env, v, o); code >= 0 {
			return code
		}
	case "package list", "package upload", "package delete":
		if code := packageCheck(env, v, &o); code >= 0 {
			return code
		}
	case "pr bind":
		if len(o.pos) > 1 || (len(o.pos) == 1 && (o.flags["branch"] != "" || !isNum(o.pos[0]))) {
			return usage(env, v)
		}
	case "workflow dispatch":
		if len(o.pos) != 1 || !workflowFile.MatchString(o.pos[0]) || !refOK(o.flags["ref"]) {
			return usage(env, v)
		}
	case "protection list":
		if len(o.pos) > 1 || (len(o.pos) == 1 && !branchName.MatchString(o.pos[0])) {
			return usage(env, v)
		}
	case "secret list":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	case "secret set":
		if code := secretSetCheck(env, v, &o); code >= 0 {
			return code
		}
	case "secret delete":
		if len(o.pos) != 1 || !secretName.MatchString(o.pos[0]) {
			return usage(env, v)
		}
	case "runners list":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	case "label list":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	case "label create":
		if code := labelCreateCheck(env, v, o); code >= 0 {
			return code
		}
	case "label rename":
		if len(o.pos) != 2 || !isNum(o.pos[0]) || !oneLine(o.pos[1]) {
			return usage(env, v)
		}
	case "board list":
		if len(o.pos) != 0 || (o.flags["state"] != "" && o.flags["state"] != "open" && o.flags["state"] != "closed") {
			return usage(env, v)
		}
	case "board view":
		if len(o.pos) != 1 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
	case "board add", "board move":
		skip := 1
		if v.Name == "move" {
			skip = 2
		}
		if len(o.pos) < skip+1 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
		for _, a := range o.pos[skip:] {
			if !isNum(a) {
				return usage(env, v)
			}
		}
	case "pr edit":
		if len(o.pos) != 1 || (o.flags["body"] != "" && o.flags["body-file"] != "") || (o.flags["title"] == "" && o.flags["body"] == "" && o.flags["body-file"] == "" && o.flags["base"] == "") {
			return usage(env, v)
		}
		var err error
		if n, err = strconv.Atoi(o.pos[0]); err != nil || n < 1 {
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
		if r := guard.Credit(o.flags["title"] + "\n" + o.flags["body"]); len(r) > 0 {
			fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r[0])
			return ExitLint
		}
		if o.flags["title"] != "" {
			if code := lintText(env, o.flags["title"]); code != ExitOK {
				return code
			}
		}
		if o.flags["body"] != "" {
			if code := lintText(env, o.flags["body"]); code != ExitOK {
				return code
			}
			o.flags["body"] = stampTrace(env, o.flags, o.flags["body"])
		}
	case "issue edit":
		if len(o.pos) != 1 || !isNum(o.pos[0]) || issueEditEmpty(o) || (o.flags["body"] != "" && o.flags["body-file"] != "") {
			return usage(env, v)
		}
		if s := o.flags["state"]; s != "" && s != "open" && s != "closed" {
			return usage(env, v)
		}
		n, _ = strconv.Atoi(o.pos[0])
		if f := o.flags["body-file"]; f != "" {
			raw, err := readText(env, f)
			if err != nil {
				fmt.Fprintf(env.Stderr, "vcs: cannot read a body from %s\n", f)
				return ExitUsage
			}
			o.flags["body"] = raw
		}
		if r := guard.Credit(o.flags["title"] + "\n" + o.flags["body"]); len(r) > 0 {
			fmt.Fprintf(env.Stderr, "vcs: guard: %s\n", r[0])
			return ExitLint
		}
		for _, text := range []string{o.flags["title"], o.flags["body"]} {
			if text == "" {
				continue
			}
			if code := lintText(env, text); code != ExitOK {
				return code
			}
		}
		o.flags["body"] = stampTrace(env, o.flags, o.flags["body"])
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
		o.flags["body"] = stampTrace(env, o.flags, o.flags["body"])
	case "comment list", "comment delete":
		if len(o.pos) != 1 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
		n, _ = strconv.Atoi(o.pos[0])
	case "comment add", "comment edit":
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
		o.flags["body"] = stampTrace(env, o.flags, o.flags["body"])
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
	case "ci artifact get":
		if len(o.pos) < 1 || len(o.pos) > 2 || (o.flags["all"] == "true") != (len(o.pos) == 1) {
			return usage(env, v)
		}
		runArg = o.pos[0]
		if o.flags["all"] == "true" {
			if st, err := os.Stat(o.flags["o"]); o.flags["o"] != "" && (err != nil || !st.IsDir()) {
				fmt.Fprintf(env.Stderr, "vcs: %s is not a directory\n", o.flags["o"])
				return ExitUsage
			}
		} else if name := o.pos[1]; name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
			fmt.Fprintln(env.Stderr, "vcs: an artifact name has no path separator")
			return ExitUsage
		}
	case "ci jobs", "ci rerun", "ci artifacts":
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
	if errors.Is(err, forge.ErrUnsupported) {
		return ExitUnavailable
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

// commentLine puts a comment on one line. Each newline becomes the two characters \n.
func commentLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.NewReplacer("\n", `\n`, "\r", " ").Replace(s)
}

func row(env Env, cols ...string) {
	for i := range cols {
		cols[i] = field(cols[i])
	}
	fmt.Fprintln(env.Stdout, strings.Join(cols, "\t"))
}

// refuseClosed reads the state of a pull request. For a closed or merged one it prints a message
// and returns ExitFail, so the verb never reports a change that did not happen.
func refuseClosed(ctx context.Context, c forge.Client, env Env, verb string, n int) int {
	s, err := c.PRStatus(ctx, n)
	if err != nil {
		return fail(env, err)
	}
	if s.State == "closed" || s.State == "merged" {
		fmt.Fprintf(env.Stderr, "vcs: pull request %d is %s, so pr %s changes nothing\n", n, s.State, verb)
		return ExitFail
	}
	return ExitOK
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
		cols := []string{s.State, strconv.FormatBool(s.Draft), strconv.FormatBool(s.Mergeable),
			strings.Join(s.ConflictedFiles, ","), s.CI, s.HeadSHA, s.URL}
		if o.flags["pushed"] == "true" {
			ct, ok := c.(forge.CommitTimer)
			if !ok {
				fmt.Fprintln(env.Stderr, "vcs: this driver cannot read the commit time")
				return ExitUnavailable
			}
			when, err := ct.CommitTime(ctx, s.HeadSHA)
			if err != nil {
				return fail(env, err)
			}
			cols = append(cols, when)
		}
		row(env, cols...)
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
		// The server ties an artifact to the run. A host without artifacts adds no row.
		arts, err := c.CIArtifacts(ctx, id)
		if err != nil && !errors.Is(err, forge.ErrUnsupported) {
			fmt.Fprintf(env.Stderr, "vcs: the artifact list failed: %v\n", err)
		}
		for _, a := range arts {
			row(env, append([]string{"artifact"}, artifactCols(a)...)...)
		}
	case "ci artifacts":
		id, err := resolveRun(ctx, c, runIndex)
		if err != nil {
			return fail(env, err)
		}
		arts, err := c.CIArtifacts(ctx, id)
		if err != nil {
			return fail(env, err)
		}
		for _, a := range arts {
			row(env, artifactCols(a)...)
		}
	case "ci artifact get":
		return artifactGet(ctx, c, env, o, runIndex)
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
			if _, _, err := c.Comment(ctx, n, b.String()); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "comment posted")
		}
	case "issue create":
		num, addr, err := c.CreateIssue(ctx, o.pos[0], o.flags["body"])
		if err != nil {
			return fail(env, err)
		}
		row(env, strconv.Itoa(num), addr)
	case "pr ready":
		if code := refuseClosed(ctx, c, env, v.Name, n); code != ExitOK {
			return code
		}
		if err := c.PRReady(ctx, n); err != nil {
			return fail(env, err)
		}
		row(env, "ready", strconv.Itoa(n))
	case "pr draft", "pr close", "pr rebase":
		sc, ok := c.(forge.PRStateChanger)
		if !ok {
			fmt.Fprintf(env.Stderr, "vcs: this host has no pr %s yet\n", v.Name)
			return ExitUnavailable
		}
		switch v.Name {
		case "draft":
			if code := refuseClosed(ctx, c, env, v.Name, n); code != ExitOK {
				return code
			}
			if err := sc.PRDraft(ctx, n); err != nil {
				return fail(env, err)
			}
			row(env, "draft", strconv.Itoa(n))
		case "close":
			if err := sc.PRClose(ctx, n); err != nil {
				return fail(env, err)
			}
			row(env, "closed", strconv.Itoa(n))
		case "rebase":
			ticked, err := sc.PRRebase(ctx, n)
			if err != nil {
				return fail(env, err)
			}
			if ticked {
				fmt.Fprintln(env.Stdout, "ok")
			} else {
				fmt.Fprintln(env.Stdout, "already")
			}
		}
	case "pr list":
		rd, ok := c.(forge.Reader)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no pr list yet")
			return ExitUnavailable
		}
		prs, err := rd.PRList(ctx, listFilter(o, limit))
		if err != nil {
			return fail(env, err)
		}
		for _, p := range prs {
			ci := ""
			if o.flags["ci"] == "true" {
				s, err := c.PRStatus(ctx, p.Number)
				if err != nil {
					return fail(env, err)
				}
				ci = s.CI
			}
			row(env, strconv.Itoa(p.Number), p.Title, p.Head, p.Base, strconv.FormatBool(p.Mergeable), ci,
				strings.Join(p.Labels, ","), p.Milestone)
		}
	case "issue list":
		rd, ok := c.(forge.Reader)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no issue list yet")
			return ExitUnavailable
		}
		is, err := rd.IssueList(ctx, listFilter(o, limit))
		if err != nil {
			return fail(env, err)
		}
		for _, i := range is {
			row(env, strconv.Itoa(i.Number), i.Title, i.State, strings.Join(i.Labels, ","), i.Milestone)
		}
	case "issue view":
		rd, ok := c.(forge.Reader)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no issue view yet")
			return ExitUnavailable
		}
		i, err := rd.IssueView(ctx, n)
		if err != nil {
			return fail(env, err)
		}
		id := ""
		if i.ID != 0 {
			id = strconv.FormatInt(i.ID, 10)
		}
		row(env, strconv.Itoa(i.Number), i.Title, i.State, strings.Join(i.Labels, ","), i.Milestone, id, i.URL)
		text := func(s string) string { return strings.TrimRight(strings.ReplaceAll(s, "\r", ""), "\n") }
		fmt.Fprintf(env.Stdout, "\n%s\n", text(i.Body))
		for _, cm := range i.Comments {
			fmt.Fprintf(env.Stdout, "\ncomment\t%s\t%s\n%s\n", field(cm.Author), field(cm.Created), text(cm.Body))
		}
	case "ci failed":
		return ciFailed(ctx, c, env, o.pos[0])
	case "pr bind":
		return prBind(ctx, c, env, o)
	case "package list", "package upload", "package delete":
		return packageRun(ctx, c, env, v, o)
	case "pr statuses":
		return prStatuses(ctx, c, env, o)
	case "issue newer", "issue top":
		nw, ok := c.(forge.Newer)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no issue index list yet")
			return ExitUnavailable
		}
		if v.Name == "top" {
			top, err := nw.TopIndex(ctx)
			if err != nil {
				return fail(env, err)
			}
			row(env, strconv.Itoa(top))
			break
		}
		mark, _ := strconv.Atoi(o.pos[0])
		state := o.flags["state"]
		if state == "" {
			state = "open"
		}
		items, err := nw.IssuesNewer(ctx, mark, state, o.flags["kind"] == "any")
		if err != nil {
			return fail(env, err)
		}
		for _, it := range items {
			row(env, strconv.Itoa(it.Number), it.Kind, it.Author, it.Title)
		}
	case "issue timeline":
		t, ok := c.(forge.Timeliner)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no issue timeline yet")
			return ExitUnavailable
		}
		evs, err := t.IssueTimeline(ctx, n)
		if err != nil {
			return fail(env, err)
		}
		for _, e := range evs {
			row(env, strconv.FormatInt(e.ID, 10), e.Created, e.Type, e.User, e.Detail)
		}
	case "pr heal":
		return heal(ctx, c, env, o)
	case "pr tick":
		return tick(ctx, c, env, o.flags["dry-run"] == "true")
	case "pr merge":
		m, ok := c.(forge.PRMerger)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no pr merge yet")
			return ExitUnavailable
		}
		if err := m.PRMerge(ctx, n, forge.PRMerge{Method: o.flags["method"], DeleteBranch: o.flags["delete-branch"] != ""}); err != nil {
			return fail(env, err)
		}
		row(env, "merged", strconv.Itoa(n))
	case "pr edit":
		ed, ok := c.(forge.PREditor)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no pr edit yet")
			return ExitUnavailable
		}
		var in forge.PREdit
		if t := o.flags["title"]; t != "" {
			in.Title = &t
		}
		if b := o.flags["body"]; b != "" {
			in.Body = &b
		}
		if b := o.flags["base"]; b != "" {
			in.Base = &b
		}
		if err := ed.PREdit(ctx, n, in); err != nil {
			return fail(env, err)
		}
		row(env, "edited", strconv.Itoa(n))
	case "milestone list", "milestone view", "milestone create":
		mm, ok := c.(forge.MilestoneManager)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no milestone support yet")
			return ExitUnavailable
		}
		switch v.Name {
		case "list":
			state := o.flags["state"]
			if state == "" {
				state = "open"
			}
			ms, err := mm.MilestoneList(ctx, state)
			if err != nil {
				return fail(env, err)
			}
			for _, m := range ms {
				row(env, strconv.FormatInt(m.ID, 10), m.Title, m.State, strconv.Itoa(m.Open), strconv.Itoa(m.Closed), m.Due)
			}
		case "view":
			m, err := mm.MilestoneView(ctx, o.pos[0])
			if err != nil {
				return fail(env, err)
			}
			desc := strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(m.Description)
			row(env, strconv.FormatInt(m.ID, 10), m.Title, m.State, strconv.Itoa(m.Open), strconv.Itoa(m.Closed), m.Due, desc)
		case "create":
			if err := mm.MilestoneCreate(ctx, o.pos[0], o.flags["due"], o.flags["desc"]); err != nil {
				return fail(env, err)
			}
			row(env, "created", o.pos[0])
		}
	case "issue edit":
		ed, ok := c.(forge.IssueEditor)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no issue edit yet")
			return ExitUnavailable
		}
		if err := ed.IssueEdit(ctx, n, issueEditInput(o)); err != nil {
			return fail(env, err)
		}
		row(env, "edited", strconv.Itoa(n))
	case "time add", "time list", "time total", "stopwatch list":
		tt, ok := c.(forge.TimeTracker)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no time tracking yet")
			return ExitUnavailable
		}
		switch v.Noun + " " + v.Name {
		case "time add":
			n, _ := strconv.Atoi(o.pos[0])
			secs, _ := parseDuration(o.pos[1])
			total, err := tt.TimeAdd(ctx, n, secs)
			if err != nil {
				return fail(env, err)
			}
			if total < 0 {
				fmt.Fprintln(env.Stderr, "vcs: the time is logged, but the new total could not be read")
				row(env, "ok")
			} else {
				row(env, "ok", strconv.FormatInt(total, 10), hms(total))
			}
		case "time list", "time total":
			n := 0
			if len(o.pos) == 1 {
				n, _ = strconv.Atoi(o.pos[0])
			}
			es, err := tt.TimeList(ctx, n, o.flags["mine"] == "true")
			if err != nil {
				return fail(env, err)
			}
			if v.Name == "total" {
				var total int64
				for _, e := range es {
					total += e.Seconds
				}
				row(env, strconv.FormatInt(total, 10), hms(total))
				break
			}
			for _, e := range es {
				issue := ""
				if e.Issue > 0 {
					issue = strconv.Itoa(e.Issue)
				}
				row(env, strconv.FormatInt(e.ID, 10), e.User, strconv.FormatInt(e.Seconds, 10), e.Created, issue)
			}
		case "stopwatch list":
			ws, err := tt.StopwatchList(ctx)
			if err != nil {
				return fail(env, err)
			}
			for _, w := range ws {
				row(env, w.Repo, strconv.Itoa(w.Issue), strconv.FormatInt(w.Seconds, 10), w.Created)
			}
			if o.flags["check"] == "true" && len(ws) > 0 {
				return ExitFound
			}
		}
	case "workflow dispatch":
		wd, ok := c.(forge.WorkflowDispatcher)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no workflow dispatch yet")
			return ExitUnavailable
		}
		if err := wd.WorkflowDispatch(ctx, o.pos[0], o.flags["ref"]); err != nil {
			return fail(env, err)
		}
		fmt.Fprintln(env.Stdout, "ok")
	case "protection list":
		pl, ok := c.(forge.ProtectionLister)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no protection list yet")
			return ExitUnavailable
		}
		ps, err := pl.ProtectionList(ctx)
		if err != nil {
			return fail(env, err)
		}
		for _, p := range ps {
			if len(o.pos) == 1 && !protectionCovers(p.Rule, o.pos[0]) {
				continue
			}
			admins := "no"
			if p.Admins {
				admins = "yes"
			}
			row(env, p.Rule, strings.Join(p.Contexts, ","), strconv.Itoa(p.Approvals), p.Push, admins)
		}
	case "secret list", "secret set", "secret delete":
		sm, ok := c.(forge.SecretManager)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no secret verbs yet")
			return ExitUnavailable
		}
		switch v.Name {
		case "list":
			ss, err := sm.SecretList(ctx)
			if err != nil {
				return fail(env, err)
			}
			for _, s := range ss {
				row(env, s.Name, s.Created)
			}
		case "set":
			if err := sm.SecretSet(ctx, o.pos[0], o.flags["value"]); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "ok")
		default:
			if err := sm.SecretDelete(ctx, o.pos[0]); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "ok")
		}
	case "runners list":
		rl, ok := c.(forge.RunnerLister)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no runner list yet")
			return ExitUnavailable
		}
		rs, err := rl.RunnerList(ctx)
		if err != nil {
			return fail(env, err)
		}
		for _, r := range rs {
			row(env, strconv.FormatInt(r.ID, 10), r.Name, r.Status, strings.Join(r.Labels, ","), r.UUID, r.Version)
		}
	case "label list", "label create", "label rename":
		lm, ok := c.(forge.LabelManager)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no label support yet")
			return ExitUnavailable
		}
		switch v.Name {
		case "list":
			ls, err := lm.LabelList(ctx)
			if err != nil {
				return fail(env, err)
			}
			for _, l := range ls {
				row(env, strconv.FormatInt(l.ID, 10), l.Name, l.Color, l.Description)
			}
		case "create":
			if err := lm.LabelCreate(ctx, o.pos[0], o.pos[1], o.flags["desc"]); err != nil {
				return fail(env, err)
			}
			row(env, "created", o.pos[0])
		case "rename":
			id, _ := strconv.ParseInt(o.pos[0], 10, 64)
			if err := lm.LabelRename(ctx, id, o.pos[1]); err != nil {
				return fail(env, err)
			}
			row(env, "renamed", o.pos[0], o.pos[1])
		}
	case "board list":
		bs, err := c.BoardList(ctx, o.flags["state"] == "closed")
		if err != nil {
			return fail(env, err)
		}
		for _, b := range bs {
			row(env, b.ID, b.Title, b.URL)
		}
	case "board view":
		cs, err := c.BoardView(ctx, o.pos[0])
		if err != nil {
			return fail(env, err)
		}
		for _, k := range cs {
			row(env, k.ColumnID, k.Column, k.ItemID, strconv.Itoa(k.Number), k.Title)
		}
	case "board add", "board move":
		skip := 1
		if v.Name == "move" {
			skip = 2
		}
		var nums []int
		for _, a := range o.pos[skip:] {
			x, _ := strconv.Atoi(a)
			nums = append(nums, x)
		}
		var err error
		if v.Name == "add" {
			err = c.BoardAdd(ctx, o.pos[0], nums)
		} else {
			err = c.BoardMove(ctx, o.pos[0], o.pos[1], nums)
		}
		if err != nil {
			return fail(env, err)
		}
		fmt.Fprintln(env.Stdout, "ok")
	case "pr create":
		num, addr, err := c.CreatePR(ctx, forge.NewPR{Title: o.pos[0], Body: o.flags["body"], Head: o.flags["head"], Base: o.flags["base"], Draft: o.flags["draft"] == "true"})
		if err != nil {
			return fail(env, err)
		}
		row(env, strconv.Itoa(num), addr)
	case "comment add":
		id, addr, err := c.Comment(ctx, n, o.flags["body"])
		if err != nil {
			return fail(env, err)
		}
		row(env, strconv.FormatInt(id, 10), addr)
	case "comment list", "comment edit", "comment delete":
		cm, ok := c.(forge.CommentManager)
		if !ok {
			fmt.Fprintln(env.Stderr, "vcs: this host has no comment support yet")
			return ExitUnavailable
		}
		switch v.Name {
		case "list":
			rows, err := cm.CommentList(ctx, n)
			if err != nil {
				return fail(env, err)
			}
			for _, r := range rows {
				row(env, strconv.FormatInt(r.ID, 10), r.Author, r.Created, commentLine(r.Body), r.Updated)
			}
		case "edit":
			if err := cm.CommentEdit(ctx, int64(n), o.flags["body"]); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "ok")
		case "delete":
			if err := cm.CommentDelete(ctx, int64(n)); err != nil {
				return fail(env, err)
			}
			fmt.Fprintln(env.Stdout, "ok")
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
		if len(o.pos) != 0 {
			return usage(env, v)
		}
		if user == "" || pass == "" {
			if !env.IsTerminal || env.ReadPassword == nil {
				fmt.Fprintln(env.Stderr, "vcs: give --user, and set VCS_PASSWORD")
				return usage(env, v)
			}
			var ok bool
			if user, pass, ok = promptLogin(env, host, user, pass); !ok {
				return ExitFail
			}
		}
		name, val, err := session.Login(ctx, base, user, pass)
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		if err := session.Save(env.ConfigDir, host, name, val); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
	case "import":
		data, err := io.ReadAll(io.LimitReader(env.Stdin, 4096))
		val := strings.Join(strings.Fields(string(data)), "")
		if err != nil || val == "" {
			return usage(env, v)
		}
		// The input is the value alone, or name=value. A bare value goes under each known name.
		name := ""
		if n, rest, ok := strings.Cut(val, "="); ok && n != "" && !strings.Contains(n, ";") {
			name, val = n, rest
		}
		if name == "" {
			name, err = session.CheckAny(ctx, base, val)
		} else {
			err = session.Check(ctx, base, name, val)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		if err := session.Save(env.ConfigDir, host, name, val); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
	case "status":
		name, val, err := session.Load(env.ConfigDir, host)
		if err == nil {
			err = session.Check(ctx, base, name, val)
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

func artifactCols(a forge.Artifact) []string {
	note := ""
	if a.Expired {
		note = "expired"
	}
	return []string{strconv.FormatInt(a.ID, 10), a.Name, strconv.FormatInt(a.Size, 10), note}
}

// artifactGet writes each wanted artifact as the zip the server sent. It writes no other file.
func artifactGet(ctx context.Context, c forge.Client, env Env, o opts, runIndex int64) int {
	id, err := resolveRun(ctx, c, runIndex)
	if err != nil {
		return fail(env, err)
	}
	arts, err := c.CIArtifacts(ctx, id)
	if err != nil {
		return fail(env, err)
	}
	all := o.flags["all"] == "true"
	var want []forge.Artifact
	if all {
		for _, a := range arts {
			if !a.Expired {
				want = append(want, a)
			}
		}
		if len(want) == 0 {
			fmt.Fprintf(env.Stderr, "vcs: run %d has no artifact that can be downloaded\n", id)
			return ExitFail
		}
	} else {
		name := o.pos[1]
		for _, a := range arts {
			if a.Name == name {
				want = append(want, a)
			}
		}
		if len(want) == 0 {
			names := make([]string, 0, len(arts))
			for _, a := range arts {
				names = append(names, a.Name)
			}
			fmt.Fprintf(env.Stderr, "vcs: run %d has no artifact named %q. It has: %s\n", id, name, strings.Join(names, ", "))
			return ExitFail
		}
		if want[0].Expired {
			fmt.Fprintf(env.Stderr, "vcs: artifact %q expired and the server removed it\n", name)
			return ExitFail
		}
	}
	for _, a := range want {
		path, overwrite := a.Name+".zip", false
		switch out := o.flags["o"]; {
		case all && out != "":
			path = filepath.Join(out, path)
		case !all && out != "":
			path, overwrite = out, true
		}
		flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if overwrite {
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		f, err := os.OpenFile(path, flag, 0o644)
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: %v\n", err)
			return ExitFail
		}
		n, err := c.CIArtifactGet(ctx, a, f)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(path) // never leave a half file
			return fail(env, err)
		}
		row(env, path, strconv.FormatInt(n, 10))
	}
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

func isNum(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0
}

// listFilter builds the filter of a list verb from its flags. --label takes names separated
// by commas, and an item must carry every one.
func listFilter(o opts, limit int) forge.ListFilter {
	f := forge.ListFilter{State: o.flags["state"], Milestone: o.flags["milestone"], Query: o.flags["query"], Limit: limit}
	for _, l := range strings.Split(o.flags["label"], ",") {
		if l = strings.TrimSpace(l); l != "" {
			f.Labels = append(f.Labels, l)
		}
	}
	return f
}

// ciFailed prints one row for each red job of a pull request head or a commit: run id, job id,
// job name, and the first error line of the log. It checks that each run belongs to the commit.
func ciFailed(ctx context.Context, c forge.Client, env Env, target string) int {
	sha := target
	if len(target) <= 6 {
		n, _ := strconv.Atoi(target)
		s, err := c.PRStatus(ctx, n)
		if err != nil {
			return fail(env, err)
		}
		sha = s.HeadSHA
	}
	runs, err := c.CIRuns(ctx, forge.RunFilter{SHA: sha, Limit: 0})
	if err != nil {
		return fail(env, err)
	}
	for _, r := range runs {
		if !strings.HasPrefix(r.SHA, strings.ToLower(sha)) && !strings.HasPrefix(strings.ToLower(sha), strings.ToLower(r.SHA)) {
			fmt.Fprintf(env.Stderr, "vcs: run %d belongs to commit %s, not %s\n", r.ID, r.SHA, sha)
			return ExitFail
		}
		if r.Status != "failure" && r.Status != "error" {
			continue
		}
		jobs, err := c.CIJobs(ctx, r.ID)
		if err != nil {
			return fail(env, err)
		}
		for _, j := range jobs {
			if j.Status != "failure" && j.Status != "error" {
				continue
			}
			first := ""
			if !j.NoLog {
				log, err := c.CILog(ctx, j.ID)
				switch {
				case err != nil:
					first = "(log unreadable)"
				default:
					first = firstError(log)
				}
			}
			row(env, strconv.FormatInt(r.ID, 10), strconv.FormatInt(j.ID, 10), j.Name, first)
		}
	}
	return ExitOK
}

// firstError returns the text of the first ::error:: line of a log.
func firstError(log string) string {
	for _, line := range strings.Split(log, "\n") {
		if i := strings.Index(line, "::error"); i >= 0 {
			rest := line[i:]
			if j := strings.Index(rest, "::"); j >= 0 {
				if k := strings.Index(rest[j+2:], "::"); k >= 0 {
					return strings.TrimSpace(rest[j+2+k+2:])
				}
			}
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// prStatuses prints the status rows of one commit, newest first: context, state, address. With
// --latest it keeps the newest row of each context, sorted by context. With --description it adds
// the description as a last column.
func prStatuses(ctx context.Context, c forge.Client, env Env, o opts) int {
	l, ok := c.(forge.StatusLister)
	if !ok {
		fmt.Fprintln(env.Stderr, "vcs: this host has no pr statuses yet")
		return ExitUnavailable
	}
	rows, err := l.CommitStatuses(ctx, o.pos[0])
	if err != nil {
		return fail(env, err)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	if o.flags["latest"] == "true" {
		seen := map[string]bool{}
		kept := rows[:0:0]
		for _, r := range rows {
			if !seen[r.Context] {
				seen[r.Context] = true
				kept = append(kept, r)
			}
		}
		sort.SliceStable(kept, func(i, j int) bool { return kept[i].Context < kept[j].Context })
		rows = kept
	}
	for _, r := range rows {
		cols := []string{r.Context, r.Status, r.URL}
		if o.flags["description"] == "true" {
			cols = append(cols, r.Description)
		}
		row(env, cols...)
	}
	return ExitOK
}

// promptLogin asks for what a login still lacks. It says what the login is for, offers a known
// user name, and reads the password without an echo. It never prints the password.
func promptLogin(env Env, host, user, pass string) (string, string, bool) {
	fmt.Fprintf(env.Stderr, "This signs in to the Forgejo web interface of %s.\n"+
		"The token API cannot do some verbs, such as `vcs ci rerun`. They need a web session.\n"+
		"vcs keeps the session cookie, never the password.\n", host)
	if user == "" {
		def := ""
		if env.KnownUser != nil {
			def = env.KnownUser(host)
		}
		if def != "" {
			fmt.Fprintf(env.Stderr, "User name [%s]: ", def)
		} else {
			fmt.Fprint(env.Stderr, "User name: ")
		}
		line, _ := bufio.NewReader(env.Stdin).ReadString('\n')
		user = strings.TrimSpace(line)
		if user == "" {
			user = def
		}
		if user == "" {
			fmt.Fprintln(env.Stderr, "vcs: no user name given")
			return "", "", false
		}
	}
	if pass == "" {
		fmt.Fprintf(env.Stderr, "Password for %s: ", user)
		p, err := env.ReadPassword()
		fmt.Fprintln(env.Stderr)
		if err != nil || p == "" {
			fmt.Fprintln(env.Stderr, "vcs: no password given")
			return "", "", false
		}
		pass = p
	}
	return user, pass, true
}
