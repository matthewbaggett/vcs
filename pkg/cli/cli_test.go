package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/lint"
)

type fake struct {
	rerun   *[]string
	assets  []forge.Asset
	comment *string
	pr      forge.PRStatus
	runs    []forge.Run
	jobs    []forge.Job
	arts    []forge.Artifact
	artData string
	log     string
	err     error
}

func (f fake) PRStatus(context.Context, int) (forge.PRStatus, error) { return f.pr, f.err }
func (f fake) CIRuns(context.Context, forge.RunFilter) ([]forge.Run, error) {
	return f.runs, f.err
}
func (f fake) CIJobs(context.Context, int64) ([]forge.Job, error)           { return f.jobs, f.err }
func (f fake) CIArtifacts(context.Context, int64) ([]forge.Artifact, error) { return f.arts, f.err }
func (f fake) CIArtifactGet(_ context.Context, _ forge.Artifact, w io.Writer) (int64, error) {
	n, err := io.WriteString(w, f.artData)
	return int64(n), err
}
func (f fake) CIRerun(_ context.Context, id int64, failed bool) error {
	if f.rerun != nil {
		*f.rerun = append(*f.rerun, fmt.Sprintf("%d/%v", id, failed))
	}
	return f.err
}
func (f fake) Attach(_ context.Context, _ int, files []string) ([]forge.Asset, error) {
	return f.assets, f.err
}
func (f fake) Comment(_ context.Context, _ int, body string) error {
	if f.comment != nil {
		*f.comment = body
	}
	return f.err
}
func (f fake) CreatePR(_ context.Context, in forge.NewPR) (int, string, error) {
	if f.comment != nil {
		*f.comment = in.Head + "<-" + in.Base + "|" + in.Title + "|" + in.Body
	}
	return 7, "https://git.example/o/r/pulls/7", f.err
}
func (f fake) CreateIssue(_ context.Context, title, body string) (int, string, error) {
	if f.comment != nil {
		*f.comment = title + "|" + body
	}
	return 8, "https://git.example/o/r/issues/8", f.err
}
func (f fake) PRReady(_ context.Context, n int) error {
	if f.comment != nil {
		*f.comment = fmt.Sprintf("ready %d", n)
	}
	return f.err
}
func (f fake) PREdit(_ context.Context, n int, in forge.PREdit) error {
	if f.comment != nil {
		t, b := "-", "-"
		if in.Title != nil {
			t = *in.Title
		}
		if in.Body != nil {
			b = *in.Body
		}
		*f.comment = fmt.Sprintf("edit %d|%s|%s", n, t, b)
	}
	return f.err
}
func (f fake) PRMerge(_ context.Context, n int, in forge.PRMerge) error {
	if f.comment != nil {
		*f.comment = fmt.Sprintf("merge %d|%s|%v", n, in.Method, in.DeleteBranch)
	}
	return f.err
}
func (f fake) BoardList(_ context.Context, closed bool) ([]forge.Board, error) {
	return []forge.Board{{ID: "1", Title: "Plan", URL: "https://git.example/o/r/projects/1"}}, f.err
}
func (f fake) BoardView(context.Context, string) ([]forge.Card, error) {
	return []forge.Card{{ColumnID: "5", Column: "Doing", ItemID: "90", Number: 3, Title: "A card"}}, f.err
}
func (f fake) BoardAdd(_ context.Context, b string, n []int) error {
	if f.comment != nil {
		*f.comment = fmt.Sprintf("add %s %v", b, n)
	}
	return f.err
}
func (f fake) BoardMove(_ context.Context, b, col string, n []int) error {
	if f.comment != nil {
		*f.comment = fmt.Sprintf("move %s %s %v", b, col, n)
	}
	return f.err
}
func (f fake) CILog(context.Context, int64) (string, error) { return f.log, f.err }

func run(t *testing.T, c forge.Client, env map[string]string, tty bool, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	e := Env{
		Stdout: &out, Stderr: &errb, IsTerminal: tty,
		Getenv:    func(k string) string { return env[k] },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(host, repo string) (forge.Client, error) {
			if host == "github.com" {
				return nil, forge.ErrUnsupported
			}
			return c, nil
		},
	}
	code := Run(context.Background(), args, e)
	return code, out.String(), errb.String()
}

func TestPRStatusRow(t *testing.T) {
	c := fake{pr: forge.PRStatus{State: "open", Draft: true, CI: "failure", HeadSHA: "abc", URL: "u", ConflictedFiles: []string{"a", "b"}}}
	code, out, errs := run(t, c, nil, true, "pr", "status", "12")
	if code != 0 || out != "open\ttrue\tfalse\ta,b\tfailure\tabc\tu\n" || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestAgentHintOnlyForAgent(t *testing.T) {
	c := fake{pr: forge.PRStatus{State: "open"}}
	_, _, human := run(t, c, nil, true, "pr", "status", "12")
	if strings.Contains(human, "vcs: next:") {
		t.Errorf("a person saw a hint: %q", human)
	}
	for name, env := range map[string]map[string]string{
		"flag-env":   {"VCS_CALLER": "agent"},
		"claudecode": {"CLAUDECODE": "1"},
	} {
		_, _, errs := run(t, c, env, true, "pr", "status", "12")
		if !strings.Contains(errs, "vcs: next: ") {
			t.Errorf("%s: no hint: %q", name, errs)
		}
	}
	// No terminal means an agent. VCS_CALLER=human forces the other way.
	if _, _, errs := run(t, c, nil, false, "pr", "status", "12"); !strings.Contains(errs, "vcs: next:") {
		t.Errorf("pipe: no hint: %q", errs)
	}
	if _, _, errs := run(t, c, map[string]string{"VCS_CALLER": "human"}, false, "pr", "status", "12"); errs != "" {
		t.Errorf("human forced: %q", errs)
	}
}

func TestUsageErrorsExit64(t *testing.T) {
	for _, args := range [][]string{
		{}, {"pr"}, {"pr", "status"}, {"pr", "status", "x"}, {"pr", "nope"},
		{"ci", "runs", "--limit", "x"}, {"ci", "runs", "--bogus", "1"}, {"ci", "runs", "--sha", "zz"},
		{"ci", "jobs"}, {"ci", "jobs", "abc"}, {"ci", "log", "0"}, {"ci", "log", "5", "--tail", "-1"},
		{"docs", "gen"},
		{"ci", "rerun"}, {"attach"}, {"attach", "5"}, {"attach", "x", "go.mod"}, {"attach", "5", "/no/such/file"},
		{"session", "status", "extra"}, {"session", "login"},
	} {
		if code, _, _ := run(t, fake{}, nil, true, args...); code != ExitUsage {
			t.Errorf("%v exit %d, want 64", args, code)
		}
	}
}

func TestExitCodes(t *testing.T) {
	if code, _, errs := run(t, fake{}, nil, true, "--host", "github.com", "pr", "status", "1"); code != ExitUnavailable || !strings.Contains(errs, "github.com") {
		t.Errorf("github: %d %q", code, errs)
	}
	if code, _, _ := run(t, fake{err: errors.New("boom")}, nil, true, "pr", "status", "1"); code != ExitFail {
		t.Errorf("fail: %d", code)
	}
}

func TestCIRunsAndJobs(t *testing.T) {
	c := fake{
		runs: []forge.Run{{ID: 9, Index: 4, Status: "failure", Ref: "#3", SHA: "s", Title: "t", URL: "https://git.example/o/r/actions/runs/4"}},
		jobs: []forge.Job{{ID: 7, Name: "w", Status: "success", Attempt: 1, NoLog: true}},
	}
	if _, out, _ := run(t, c, nil, true, "ci", "runs"); out != "9\t4\tfailure\t#3\ts\tt\thttps://git.example/o/r/actions/runs/4\n" {
		t.Errorf("runs: %q", out)
	}
	if _, out, _ := run(t, c, nil, true, "ci", "jobs", "9"); out != "7\tw\tsuccess\t1\tno-log\n" {
		t.Errorf("jobs: %q", out)
	}
	// A run address holds the index. The command finds the id.
	if code, out, _ := run(t, c, nil, true, "ci", "jobs", "https://git.example/o/r/actions/runs/4"); code != 0 || out == "" {
		t.Errorf("jobs by address: %d %q", code, out)
	}
	if code, _, errs := run(t, c, nil, true, "ci", "jobs", "https://git.example/o/r/actions/runs/99"); code != ExitFail || !strings.Contains(errs, "99") {
		t.Errorf("unknown address: %d %q", code, errs)
	}
}

func TestCILogStripsColourAndTails(t *testing.T) {
	c := fake{log: "one\n\x1b[31mtwo\x1b[0m\nthree\n"}
	if _, out, _ := run(t, c, nil, true, "ci", "log", "5", "--tail", "2"); out != "two\nthree\n" {
		t.Errorf("tail: %q", out)
	}
	if _, out, _ := run(t, c, nil, true, "ci", "log", "5"); out != "one\ntwo\nthree\n" {
		t.Errorf("all: %q", out)
	}
	code, _, errs := run(t, fake{err: forge.ErrNoLog}, nil, true, "ci", "log", "5")
	if code != ExitFail || !strings.Contains(errs, "wrapper") {
		t.Errorf("no log: %d %q", code, errs)
	}
}

// Every verb needs an agent hint for each outcome. An empty text is allowed, a missing key is not.
func TestEveryVerbHasEveryHint(t *testing.T) {
	for _, v := range Tree {
		need := []string{OutOK, OutFail, OutUsage, OutUnavailable}
		if v.Sends {
			need = append(need, OutLint)
		}
		if v.Session {
			need = append(need, OutNoSession)
		}
		for _, out := range need {
			if _, ok := v.Hints[out]; !ok {
				t.Errorf("%s %s has no hint for %q", v.Noun, v.Name, out)
			}
		}
		if v.Usage == "" || v.Summary == "" || v.Output == "" || len(v.Exits) == 0 {
			t.Errorf("%s %s is not documented", v.Noun, v.Name)
		}
	}
}

// The committed pages must match the command tree. Run `go run ./cmd/vcs docs gen docs/Commands` to fix a drift.
func TestCommittedDocsMatchTree(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "Commands")
	for name, want := range Pages() {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Errorf("docs/Commands/%s is out of date. Run: go run ./cmd/vcs docs gen docs/Commands", name)
		}
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if _, ok := Pages()[e.Name()]; !ok && e.Name() != "README.md" {
			t.Errorf("docs/Commands/%s has no verb", e.Name())
		}
	}
}

func TestRerunVerb(t *testing.T) {
	var calls []string
	c := fake{rerun: &calls, runs: []forge.Run{{ID: 9, Index: 4, URL: "https://git.example/o/r/actions/runs/4"}}}
	if code, out, _ := run(t, c, nil, true, "ci", "rerun", "9", "--failed"); code != 0 || out != "rerun\t9\n" {
		t.Errorf("rerun: %d %q", code, out)
	}
	if code, _, _ := run(t, c, nil, true, "ci", "rerun", "https://git.example/o/r/actions/runs/4"); code != 0 {
		t.Errorf("rerun by address: %d", code)
	}
	if strings.Join(calls, ",") != "9/true,9/false" {
		t.Errorf("calls = %v", calls)
	}
	code, _, errs := run(t, fake{err: forge.ErrNoSession}, map[string]string{"VCS_CALLER": "agent"}, true, "ci", "rerun", "9")
	if code != ExitNoSession || !strings.Contains(errs, "vcs: next: stop and ask the user to run vcs session login") {
		t.Errorf("no session: %d %q", code, errs)
	}
}

func TestAttachVerb(t *testing.T) {
	var body string
	file := filepath.Join(t.TempDir(), "a.png")
	os.WriteFile(file, []byte("x"), 0o600)
	c := fake{assets: []forge.Asset{{Name: "a.png", URL: "https://x/a"}}, comment: &body}
	code, out, _ := run(t, c, nil, true, "attach", "5", file, "--comment", "--title", "Proof")
	if code != 0 || out != "a.png\thttps://x/a\ncomment posted\n" || body != "Proof\n\n![a.png](https://x/a)" {
		t.Errorf("attach: %d %q %q", code, out, body)
	}
	// A symbolic link is not a plain file.
	link := filepath.Join(t.TempDir(), "l.png")
	os.Symlink(file, link)
	if code, _, _ := run(t, c, nil, true, "attach", "5", link); code != ExitUsage {
		t.Errorf("symlink: %d", code)
	}
}

func TestSessionVerbs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/login":
			r.ParseForm()
			http.SetCookie(w, &http.Cookie{Name: "i_like_gitea", Value: "good"})
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		case "/user/settings":
			if c, err := r.Cookie("i_like_gitea"); err == nil && c.Value == "good" {
				return
			}
			w.WriteHeader(http.StatusSeeOther)
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	do := func(stdin string, env map[string]string, args ...string) (int, string) {
		var out, errb bytes.Buffer
		e := Env{
			Stdout: &out, Stderr: &errb, Stdin: strings.NewReader(stdin), ConfigDir: dir, IsTerminal: true,
			Getenv:    func(k string) string { return env[k] },
			RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
			BaseURL:   func(string) string { return srv.URL },
		}
		return Run(context.Background(), args, e), out.String() + errb.String()
	}
	if code, out := do("", nil, "session", "status"); code != ExitNoSession {
		t.Errorf("status before: %d %q", code, out)
	}
	if code, out := do("", map[string]string{"VCS_PASSWORD": "pw"}, "session", "login", "--user", "me"); code != 0 || out != "ok\n" {
		t.Errorf("login: %d %q", code, out)
	}
	if code, _ := do("", nil, "session", "status"); code != 0 {
		t.Errorf("status after login: %d", code)
	}
	do("", nil, "session", "logout")
	if code, _ := do("", nil, "session", "status"); code != ExitNoSession {
		t.Errorf("status after logout: %d", code)
	}
	if code, _ := do("bad\n", nil, "session", "import"); code != ExitFail {
		t.Errorf("import bad: %d", code)
	}
	if code, _ := do(" good \n", nil, "session", "import"); code != 0 {
		t.Errorf("import good: %d", code)
	}
	if code, _ := do("", nil, "session", "login", "--user", "me"); code != ExitUsage {
		t.Errorf("login with no password: %d", code)
	}
}

func artFake() fake {
	return fake{
		runs:    []forge.Run{{ID: 9, Index: 4, URL: "https://git.example/o/r/actions/runs/4"}},
		jobs:    []forge.Job{{ID: 7, Name: "w", Status: "success", Attempt: 1}},
		arts:    []forge.Artifact{{ID: 31, Name: "coverage", Size: 2048}, {ID: 30, Name: "old", Size: 9, Expired: true}},
		artData: "ZIP",
	}
}

func TestCIArtifactsRows(t *testing.T) {
	code, out, _ := run(t, artFake(), nil, true, "ci", "artifacts", "9")
	if code != 0 || out != "31\tcoverage\t2048\t\n30\told\t9\texpired\n" {
		t.Errorf("artifacts: %d %q", code, out)
	}
	if code, out, _ := run(t, fake{}, nil, true, "ci", "artifacts", "9"); code != 0 || out != "" {
		t.Errorf("empty: %d %q", code, out)
	}
	if code, _, _ := run(t, fake{err: forge.ErrUnsupported}, nil, true, "ci", "artifacts", "9"); code != ExitUnavailable {
		t.Errorf("unsupported: %d", code)
	}
}

func TestCIJobsShowsArtifactRows(t *testing.T) {
	code, out, _ := run(t, artFake(), nil, true, "ci", "jobs", "9")
	want := "7\tw\tsuccess\t1\t\nartifact\t31\tcoverage\t2048\t\nartifact\t30\told\t9\texpired\n"
	if code != 0 || out != want {
		t.Errorf("jobs: %d %q", code, out)
	}
	// A host with no artifact route still lists the jobs.
	c := artFake()
	c.arts = nil
	if code, out, _ := run(t, c, nil, true, "ci", "jobs", "9"); code != 0 || out != "7\tw\tsuccess\t1\t\n" {
		t.Errorf("no artifacts: %d %q", code, out)
	}
}

func TestCIArtifactGet(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)

	code, out, _ := run(t, artFake(), nil, true, "ci", "artifact", "get", "9", "coverage")
	if code != 0 || out != "coverage.zip\t3\n" {
		t.Fatalf("get: %d %q", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "coverage.zip")); string(b) != "ZIP" {
		t.Errorf("file = %q", b)
	}
	// The default file is not replaced.
	if code, _, _ := run(t, artFake(), nil, true, "ci", "artifact", "get", "9", "coverage"); code != ExitFail {
		t.Errorf("replace: %d", code)
	}
	// -o names the file, and it may replace one.
	target := filepath.Join(dir, "x.bin")
	if code, out, _ := run(t, artFake(), nil, true, "ci", "artifact", "get", "9", "coverage", "-o", target); code != 0 || out != target+"\t3\n" {
		t.Errorf("-o: %d %q", code, out)
	}
	// Only the asked files exist.
	ents, _ := os.ReadDir(dir)
	if len(ents) != 2 {
		t.Errorf("files = %v", ents)
	}
}

func TestCIArtifactGetAll(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := run(t, artFake(), nil, true, "ci", "artifact", "get", "9", "--all", "-o", dir)
	want := filepath.Join(dir, "coverage.zip") + "\t3\n"
	if code != 0 || out != want {
		t.Fatalf("all: %d %q", code, out)
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 { // the expired one is skipped
		t.Errorf("files = %v", ents)
	}
}

func TestCIArtifactGetFailures(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	for name, args := range map[string][]string{
		"no such name": {"ci", "artifact", "get", "9", "nope"},
		"expired":      {"ci", "artifact", "get", "9", "old"},
	} {
		code, _, errs := run(t, artFake(), nil, true, args...)
		if code != ExitFail || errs == "" {
			t.Errorf("%s: %d %q", name, code, errs)
		}
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Errorf("a failed download wrote %v", ents)
	}
	for _, args := range [][]string{
		{"ci", "artifact"}, {"ci", "artifact", "get"}, {"ci", "artifact", "get", "9"},
		{"ci", "artifact", "get", "9", "a", "--all"}, {"ci", "artifact", "get", "9", "../x"},
		{"ci", "artifact", "get", "9", "--all", "-o", "/no/such/dir"}, {"ci", "artifacts"},
	} {
		if code, _, _ := run(t, artFake(), nil, true, args...); code != ExitUsage {
			t.Errorf("%v exit %d, want 64", args, code)
		}
	}
	if code, _, _ := run(t, fake{}, nil, true, "--host", "github.com", "ci", "artifacts", "9"); code != ExitUnavailable {
		t.Errorf("github: %d", code)
	}
}

type stubChecker struct{ fs []lint.Finding }

func (stubChecker) Name() string                           { return "stub" }
func (s stubChecker) Check(string) ([]lint.Finding, error) { return s.fs, nil }

// A lint finding stops the send: exit 65, no upload, no comment.
func TestAttachLintStopsSend(t *testing.T) {
	var body string
	file := filepath.Join(t.TempDir(), "a.png")
	os.WriteFile(file, []byte("x"), 0o600)
	c := fake{assets: []forge.Asset{{Name: "a.png", URL: "u"}}, comment: &body}
	var out, errb bytes.Buffer
	env := Env{
		Stdout: &out, Stderr: &errb,
		Getenv:    func(string) string { return "" },
		RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
		NewClient: func(string, string) (forge.Client, error) { return c, nil },
		Checkers:  []lint.Checker{stubChecker{fs: []lint.Finding{{Checker: "stub", Line: 1, Column: 1, Rule: "r", Message: "bad word"}}}},
	}
	code := Run(context.Background(), []string{"attach", "5", file, "--comment", "--title", "Proof"}, env)
	if code != ExitLint || out.String() != "" || body != "" || !strings.Contains(errb.String(), "bad word") {
		t.Fatalf("code=%d out=%q body=%q err=%q", code, out.String(), body, errb.String())
	}
	// Without --comment, no text goes out and no lint runs.
	if code := Run(context.Background(), []string{"attach", "5", file}, env); code != ExitOK {
		t.Errorf("no comment: %d", code)
	}
}

func TestCommentAdd(t *testing.T) {
	var body string
	c := fake{comment: &body}
	run := func(chk []lint.Checker, args ...string) (int, string, string) {
		var out, errb bytes.Buffer
		env := Env{
			Stdout: &out, Stderr: &errb,
			Stdin:     strings.NewReader("from stdin"),
			Getenv:    func(string) string { return "" },
			RemoteURL: func() (string, error) { return "https://git.example/o/r.git", nil },
			NewClient: func(string, string) (forge.Client, error) { return c, nil },
			Checkers:  chk,
		}
		return Run(context.Background(), args, env), out.String(), errb.String()
	}
	if code, out, _ := run(nil, "comment", "add", "5", "--body", "Plain text."); code != ExitOK || out != "comment posted\n" || body != "Plain text." {
		t.Fatalf("body: code=%d out=%q body=%q", code, out, body)
	}
	if code, _, _ := run(nil, "comment", "add", "5", "--body-file", "-"); code != ExitOK || body != "from stdin" {
		t.Fatalf("stdin: code=%d body=%q", code, body)
	}
	body = ""
	bad := []lint.Checker{stubChecker{fs: []lint.Finding{{Checker: "stub", Line: 1, Column: 1, Rule: "r", Message: "bad word"}}}}
	if code, out, errb := run(bad, "comment", "add", "5", "--body", "x"); code != ExitLint || out != "" || body != "" || !strings.Contains(errb, "bad word") {
		t.Fatalf("lint: code=%d out=%q body=%q err=%q", code, out, body, errb)
	}
	for _, a := range [][]string{
		{"comment", "add", "5"},
		{"comment", "add", "5", "--body", "a", "--body-file", "-"},
		{"comment", "add", "x", "--body", "a"},
	} {
		if code, _, _ := run(nil, a...); code != ExitUsage {
			t.Errorf("%v: code=%d", a, code)
		}
	}
}

func TestPRCreate(t *testing.T) {
	var got string
	c := fake{comment: &got}
	args := func(more ...string) []string {
		return append([]string{"pr", "create", "Add a thing"}, more...)
	}
	code, out, _ := run(t, c, nil, false, args("--head", "feature/a-thing", "--body", "Body.", "--draft")...)
	if code != ExitOK || out != "7\thttps://git.example/o/r/pulls/7\n" || got != "feature/a-thing<-main|Add a thing|Body." {
		t.Fatalf("code=%d out=%q got=%q", code, out, got)
	}
	got = ""
	for name, a := range map[string][]string{
		"main branch": args("--head", "main", "--body", "x"),
		"bad name":    args("--head", "My_Branch", "--body", "x"),
		"credit line": args("--head", "fix/a", "--body", "Co-Authored-By: Claude <noreply@anthropic.com>"),
	} {
		if code, out, errb := run(t, c, nil, false, a...); code != ExitLint || out != "" || got != "" || !strings.Contains(errb, "guard:") {
			t.Errorf("%s: code=%d out=%q got=%q err=%q", name, code, out, got, errb)
		}
	}
	if code, _, _ := run(t, c, nil, false, args("--body", "x")...); code != ExitUsage {
		t.Errorf("no head: %d", code)
	}
}

func TestIssueCreateAndReady(t *testing.T) {
	var got string
	c := fake{comment: &got}
	code, out, _ := run(t, c, nil, false, "issue", "create", "Fix it", "--body", "Body.")
	if code != ExitOK || out != "8\thttps://git.example/o/r/issues/8\n" || got != "Fix it|Body." {
		t.Fatalf("create: code=%d out=%q got=%q", code, out, got)
	}
	got = ""
	code, out, errb := run(t, c, nil, false, "issue", "create", "Fix it", "--body", "🤖 Generated with Claude")
	if code != ExitLint || out != "" || got != "" || !strings.Contains(errb, "guard:") {
		t.Errorf("credit: code=%d out=%q got=%q err=%q", code, out, got, errb)
	}
	if code, _, _ := run(t, c, nil, false, "issue", "create", "Fix it"); code != ExitUsage {
		t.Errorf("no body: %d", code)
	}
	code, out, _ = run(t, c, nil, false, "pr", "ready", "12")
	if code != ExitOK || out != "ready\t12\n" || got != "ready 12" {
		t.Errorf("ready: code=%d out=%q got=%q", code, out, got)
	}
	if code, _, _ := run(t, c, nil, false, "pr", "ready"); code != ExitUsage {
		t.Errorf("ready no n: %d", code)
	}
}

func TestPREdit(t *testing.T) {
	var got string
	c := fake{comment: &got}
	code, out, _ := run(t, c, nil, false, "pr", "edit", "12", "--title", "New title")
	if code != ExitOK || out != "edited\t12\n" || got != "edit 12|New title|-" {
		t.Fatalf("title: code=%d out=%q got=%q", code, out, got)
	}
	code, out, _ = run(t, c, nil, false, "pr", "edit", "12", "--body", "New body.")
	if code != ExitOK || out != "edited\t12\n" || got != "edit 12|-|New body." {
		t.Fatalf("body: code=%d out=%q got=%q", code, out, got)
	}
	got = ""
	if code, out, errb := run(t, c, nil, false, "pr", "edit", "12", "--body", "🤖 Generated with Claude"); code != ExitLint || out != "" || got != "" || !strings.Contains(errb, "guard:") {
		t.Errorf("credit: code=%d out=%q got=%q err=%q", code, out, got, errb)
	}
	for _, a := range [][]string{
		{"pr", "edit", "12"},
		{"pr", "edit", "x", "--title", "T"},
		{"pr", "edit", "12", "--body", "a", "--body-file", "-"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
}

func (f fake) LabelList(context.Context) ([]forge.Label, error) {
	return []forge.Label{{ID: 4, Name: "bug", Color: "d73a4a", Description: "Broken"}}, f.err
}
func (f fake) LabelCreate(_ context.Context, name, color, desc string) error {
	if f.comment != nil {
		*f.comment = "create " + name + " " + color + " " + desc
	}
	return f.err
}
func (f fake) LabelRename(_ context.Context, id int64, name string) error {
	if f.comment != nil {
		*f.comment = fmt.Sprintf("rename %d %s", id, name)
	}
	return f.err
}

func TestLabelVerbs(t *testing.T) {
	var got string
	c := fake{comment: &got}
	if code, out, _ := run(t, c, nil, false, "label", "list"); code != ExitOK || out != "4\tbug\td73a4a\tBroken\n" {
		t.Errorf("list: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "label", "create", "wip", "#aabbcc", "--desc", "Work in progress."); code != ExitOK || out != "created\twip\n" || got != "create wip #aabbcc Work in progress." {
		t.Errorf("create: %d %q %q", code, out, got)
	}
	if code, out, _ := run(t, c, nil, false, "label", "rename", "4", "defect"); code != ExitOK || out != "renamed\t4\tdefect\n" || got != "rename 4 defect" {
		t.Errorf("rename: %d %q %q", code, out, got)
	}
	got = ""
	if code, _, errb := run(t, c, nil, false, "label", "create", "wip", "aabbcc", "--desc", "🤖 Generated with Claude"); code != ExitLint || got != "" || !strings.Contains(errb, "guard:") {
		t.Errorf("credit: %d %q %q", code, got, errb)
	}
	for _, a := range [][]string{
		{"label", "list", "x"},
		{"label", "create", "wip"},
		{"label", "create", "wip", "red"},
		{"label", "create", "wip", "aabbcc", "--desc", "a\nb"},
		{"label", "rename", "x", "n"},
		{"label", "rename", "4"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, _ := run(t, fake{err: forge.ErrNotFound}, nil, false, "label", "list"); code != ExitFail {
		t.Errorf("fail: %d", code)
	}
}

func TestPRMergeVerb(t *testing.T) {
	var got string
	c := fake{comment: &got}
	if code, out, _ := run(t, c, nil, false, "pr", "merge", "12"); code != ExitOK || out != "merged\t12\n" || got != "merge 12||false" {
		t.Errorf("default: %d %q %q", code, out, got)
	}
	if code, out, _ := run(t, c, nil, false, "pr", "merge", "12", "--method", "squash", "--delete-branch"); code != ExitOK || out != "merged\t12\n" || got != "merge 12|squash|true" {
		t.Errorf("squash: %d %q %q", code, out, got)
	}
	for _, a := range [][]string{{"pr", "merge"}, {"pr", "merge", "x"}, {"pr", "merge", "12", "--method", "weird"}} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, errb := run(t, fake{err: errors.New("boom")}, nil, false, "pr", "merge", "12"); code != ExitFail || !strings.Contains(errb, "boom") {
		t.Errorf("failure: %d %q", code, errb)
	}
}

func (f fake) MilestoneList(_ context.Context, state string) ([]forge.Milestone, error) {
	if f.comment != nil {
		*f.comment = "list " + state
	}
	return []forge.Milestone{{ID: 2, Title: "v1", State: "open", Open: 3, Closed: 1, Due: "2026-12-01"}}, f.err
}
func (f fake) MilestoneView(_ context.Context, ref string) (forge.Milestone, error) {
	return forge.Milestone{ID: 2, Title: ref, State: "open", Open: 3, Closed: 1, Description: "a\nb"}, f.err
}
func (f fake) MilestoneCreate(_ context.Context, title, due, desc string) error {
	if f.comment != nil {
		*f.comment = "create " + title + "|" + due + "|" + desc
	}
	return f.err
}

func TestMilestoneVerbs(t *testing.T) {
	var got string
	c := fake{comment: &got}
	if code, out, _ := run(t, c, nil, false, "milestone", "list"); code != ExitOK || out != "2\tv1\topen\t3\t1\t2026-12-01\n" || got != "list open" {
		t.Errorf("list: %d %q %q", code, out, got)
	}
	if code, _, _ := run(t, c, nil, false, "milestone", "list", "--state", "all"); code != ExitOK || got != "list all" {
		t.Errorf("list all: %d %q", code, got)
	}
	if code, out, _ := run(t, c, nil, false, "milestone", "view", "v1"); code != ExitOK || out != "2\tv1\topen\t3\t1\t\ta\\nb\n" {
		t.Errorf("view: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "milestone", "create", "v2", "--due", "2027-01-31", "--desc", "Next release."); code != ExitOK || out != "created\tv2\n" || got != "create v2|2027-01-31|Next release." {
		t.Errorf("create: %d %q %q", code, out, got)
	}
	got = ""
	if code, _, errb := run(t, c, nil, false, "milestone", "create", "v2", "--desc", "🤖 Generated with Claude"); code != ExitLint || got != "" || !strings.Contains(errb, "guard:") {
		t.Errorf("credit: %d %q %q", code, got, errb)
	}
	for _, a := range [][]string{
		{"milestone", "list", "x"},
		{"milestone", "list", "--state", "weird"},
		{"milestone", "view"},
		{"milestone", "create"},
		{"milestone", "create", "v2", "--due", "soon"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, _ := run(t, fake{err: forge.ErrNotFound}, nil, false, "milestone", "view", "nope"); code != ExitFail {
		t.Errorf("fail: %d", code)
	}
}

func TestBoardVerbs(t *testing.T) {
	var got string
	c := fake{comment: &got}
	if code, out, _ := run(t, c, nil, false, "board", "list"); code != ExitOK || out != "1\tPlan\thttps://git.example/o/r/projects/1\n" {
		t.Errorf("list: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "board", "view", "1"); code != ExitOK || out != "5\tDoing\t90\t3\tA card\n" {
		t.Errorf("view: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "board", "add", "1", "3", "4"); code != ExitOK || out != "ok\n" || got != "add 1 [3 4]" {
		t.Errorf("add: %d %q %q", code, out, got)
	}
	if code, out, _ := run(t, c, nil, false, "board", "move", "1", "Doing", "3"); code != ExitOK || out != "ok\n" || got != "move 1 Doing [3]" {
		t.Errorf("move: %d %q %q", code, out, got)
	}
	nos := fake{err: forge.ErrNoSession}
	if code, _, _ := run(t, nos, nil, false, "board", "move", "1", "Doing", "3"); code != ExitNoSession {
		t.Errorf("no session: %d", code)
	}
	for _, a := range [][]string{{"board", "view"}, {"board", "view", "x"}, {"board", "add", "1"}, {"board", "move", "1", "Doing"}, {"board", "list", "--state", "weird"}} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
}

func (f fake) IssueEdit(_ context.Context, n int, in forge.IssueEdit) error {
	if f.comment != nil {
		p := func(s *string) string {
			if s == nil {
				return "-"
			}
			return *s
		}
		*f.comment = fmt.Sprintf("issue %d|%s|%s|%s|%s|%v|%v", n, p(in.Title), p(in.Body), p(in.State), p(in.Milestone), in.AddLabels, in.RmLabels)
	}
	return f.err
}

func TestIssueEditVerb(t *testing.T) {
	var got string
	c := fake{comment: &got}
	code, out, _ := run(t, c, nil, false, "issue", "edit", "8", "--title", "New title", "--state", "closed", "--milestone", "none", "--add-label", "a, b", "--rm-label", "c")
	if code != ExitOK || out != "edited\t8\n" || got != "issue 8|New title|-|closed|none|[a b]|[c]" {
		t.Fatalf("all flags: %d %q %q", code, out, got)
	}
	if code, _, _ := run(t, c, nil, false, "issue", "edit", "8", "--body", "New body."); code != ExitOK || got != "issue 8|-|New body.|-|-|[]|[]" {
		t.Errorf("body: %d %q", code, got)
	}
	got = ""
	if code, out, errb := run(t, c, nil, false, "issue", "edit", "8", "--body", "🤖 Generated with Claude"); code != ExitLint || out != "" || got != "" || !strings.Contains(errb, "guard:") {
		t.Errorf("credit: %d %q %q %q", code, out, got, errb)
	}
	for _, a := range [][]string{
		{"issue", "edit"},
		{"issue", "edit", "x", "--title", "T"},
		{"issue", "edit", "8"},
		{"issue", "edit", "8", "--state", "weird"},
		{"issue", "edit", "8", "--body", "a", "--body-file", "-"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
}

type timeFake struct {
	fake
	total   int64
	entries []forge.TimeEntry
	dogs    []forge.Stopwatch
	added   *string
}

func (f timeFake) TimeAdd(_ context.Context, n int, s int64) (int64, error) {
	if f.added != nil {
		*f.added = fmt.Sprintf("%d/%d", n, s)
	}
	return f.total, f.err
}
func (f timeFake) TimeList(_ context.Context, n int, mine bool) ([]forge.TimeEntry, error) {
	return f.entries, f.err
}
func (f timeFake) StopwatchList(context.Context) ([]forge.Stopwatch, error) { return f.dogs, f.err }

func TestTimeVerbs(t *testing.T) {
	var added string
	c := timeFake{
		total:   5400,
		added:   &added,
		entries: []forge.TimeEntry{{ID: 1, User: "a", Seconds: 3600, Created: "2026-01-01", Issue: 4}, {ID: 2, User: "b", Seconds: 300, Created: "2026-01-02"}},
		dogs:    []forge.Stopwatch{{Repo: "o/r", Issue: 4, Seconds: 60, Created: "2026-01-03"}},
	}
	if code, out, _ := run(t, c, nil, false, "time", "add", "4", "1h30m"); code != ExitOK || out != "ok\t5400\t1h30m0s\n" || added != "4/5400" {
		t.Errorf("add: %d %q %q", code, out, added)
	}
	c.total = -1
	if code, out, errb := run(t, c, nil, false, "time", "add", "4", "45s"); code != ExitOK || out != "ok\n" || !strings.Contains(errb, "could not be read") {
		t.Errorf("add no total: %d %q %q", code, out, errb)
	}
	if code, out, _ := run(t, c, nil, false, "time", "list", "4", "--mine"); code != ExitOK || out != "1\ta\t3600\t2026-01-01\t4\n2\tb\t300\t2026-01-02\t\n" {
		t.Errorf("list: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "time", "total"); code != ExitOK || out != "3900\t1h5m0s\n" {
		t.Errorf("total: %d %q", code, out)
	}
	if code, out, _ := run(t, c, nil, false, "stopwatch", "list"); code != ExitOK || out != "o/r\t4\t60\t2026-01-03\n" {
		t.Errorf("stopwatch: %d %q", code, out)
	}
	if code, _, _ := run(t, c, nil, false, "stopwatch", "list", "--check"); code != ExitFound {
		t.Errorf("check found: %d", code)
	}
	c.dogs = nil
	if code, _, _ := run(t, c, nil, false, "stopwatch", "list", "--check"); code != ExitOK {
		t.Errorf("check none: %d", code)
	}
	c.err = forge.ErrNotFound
	for _, a := range [][]string{{"time", "add", "4", "1h"}, {"time", "list"}, {"stopwatch", "list"}} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitFail {
			t.Errorf("%v: %d", a, code)
		}
	}
	for _, a := range [][]string{
		{"time", "add"}, {"time", "add", "x", "1h"}, {"time", "add", "4"}, {"time", "add", "4", "0s"},
		{"time", "add", "4", "soon"}, {"time", "list", "x"}, {"time", "total", "1", "2"}, {"stopwatch", "list", "3"},
	} {
		if code, _, _ := run(t, c, nil, false, a...); code != ExitUsage {
			t.Errorf("%v: %d", a, code)
		}
	}
	if code, _, errb := run(t, fake{}, nil, false, "time", "list"); code != ExitUnavailable || !strings.Contains(errb, "no time tracking") {
		t.Errorf("unavailable: %d %q", code, errb)
	}
}
