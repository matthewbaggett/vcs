package cli

// Exit codes. They follow sysexits.h where it fits.
const (
	ExitOK          = 0
	ExitFail        = 1
	ExitUsage       = 64
	ExitUnavailable = 69
	ExitNoSession   = 77
)

// Outcomes that every verb has a hint for.
const (
	OutOK          = "ok"
	OutFail        = "fail"
	OutUsage       = "usage"
	OutUnavailable = "unavailable"
	OutNoSession   = "nosession"
)

// Verb is one command of the command tree. The docs build from this table.
type Verb struct {
	Noun    string
	Name    string
	Usage   string
	Summary string
	Output  string
	Flags   []Flag
	// Session is true when the verb needs a web session.
	Session bool
	Exits   []string
	// Hints holds the agent message of each outcome. A person never sees it.
	// An empty text means the human message is enough.
	Hints map[string]string
}

// Flag is one flag of a verb.
type Flag struct {
	Name string
	Help string
}

// Tree is the command tree.
var Tree = []Verb{
	{
		Noun: "pr", Name: "status",
		Usage:   "vcs pr status <n>",
		Summary: "Print the state of one pull request on one line.",
		Output:  "One row, tab-separated: state, draft, mergeable, conflicted files, ci, head sha, url.",
		Exits:   []string{"0 printed the row", "1 the call failed or the pull request is not found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "if ci is failure, run vcs ci runs --sha <head sha> --limit 3, then vcs ci jobs <run-id>",
			OutFail:        "check the number with vcs pr list, then run the command again",
			OutUsage:       "run vcs pr status <n> with the number of one pull request",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "ci", Name: "runs",
		Usage:   "vcs ci runs [--branch <name|#n>] [--sha <sha>] [--status <status>] [--limit <n>]",
		Summary: "List CI runs, newest first.",
		Output:  "One row for each run, tab-separated: run-id, index, status, branch, head sha, title, url. A pull request run has `#<n>` in the branch column.",
		Flags: []Flag{
			{"--branch", "Keep the runs of one branch, or of one pull request as `#<n>`."},
			{"--sha", "Keep the runs of one commit. A full 40-character sha also filters on the server."},
			{"--status", "Keep one status, such as failure or success."},
			{"--limit", "Keep the newest <n> rows. The default is 20. 0 reads every run, up to 1000."},
		},
		Exits: []string{"0 printed the rows (an empty list is not an error)", "1 the call failed", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "take a run-id from the first column and run vcs ci jobs <run-id>",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "see vcs ci runs for the flags",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "ci", Name: "jobs",
		Usage:   "vcs ci jobs <run-id|run-address>",
		Summary: "List the jobs of one CI run.",
		Output:  "One row for each job, tab-separated: job-id, name, status, attempt, note. The note is `no-log` on a wrapper job. When the run made artifacts, one row follows for each artifact: `artifact`, artifact-id, name, size in bytes, note. The note is `expired` for an artifact that the server removed. The server ties an artifact to the run, not to a job, so these rows come after all the job rows.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the run is not found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "for a failed job, run vcs ci log <job-id> --tail 80. For an artifact row, run vcs ci artifact get <run-id> <name>",
			OutFail:        "get the run-id again with vcs ci runs",
			OutUsage:       "give a run-id or the web address of a run",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "ci", Name: "artifacts",
		Usage:   "vcs ci artifacts <run-id|run-address>",
		Summary: "List the artifacts of one CI run.",
		Output:  "One row for each artifact, tab-separated: artifact-id, name, size in bytes, note. The note is `expired` for an artifact that the server removed. An empty list is not an error.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the run is not found", "64 usage error", "69 no usable driver, or the host has no artifact route"},
		Hints: map[string]string{
			OutOK:          "run vcs ci artifact get <run-id> <name> to download one",
			OutFail:        "get the run-id again with vcs ci runs",
			OutUsage:       "give a run-id or the web address of a run",
			OutUnavailable: "stop and ask the user. This host has no artifact route yet",
		},
	},
	{
		Noun: "ci", Name: "artifact get",
		Usage:   "vcs ci artifact get <run-id|run-address> <name> [-o <path>]\n       vcs ci artifact get <run-id|run-address> --all [-o <dir>]",
		Summary: "Download one artifact of a CI run, or every artifact with --all. The server sends each artifact as a zip archive, and the command writes it as it is. It does not unpack it.",
		Output:  "One row for each file written, tab-separated: path, size in bytes.",
		Flags: []Flag{
			{"--all", "Download every artifact that has not expired. Give no name."},
			{"-o", "The file to write. The default is `<name>.zip` in the current directory, and the command refuses to replace a file that exists. With --all, the path is an existing directory that takes each `<name>.zip`."},
		},
		Exits: []string{"0 wrote the files", "1 the call failed, the run or the artifact is not found, the artifact expired, or a default file exists", "64 usage error", "69 no usable driver, or the host has no artifact route"},
		Hints: map[string]string{
			OutOK:          "unpack the zip with unzip into a new empty directory, then read the files",
			OutFail:        "list the names with vcs ci artifacts <run-id>, then run the command again",
			OutUsage:       "give a run-id and one name, or a run-id and --all",
			OutUnavailable: "stop and ask the user. This host has no artifact route yet",
		},
	},
	{
		Noun: "ci", Name: "log",
		Usage:   "vcs ci log <job-id> [--tail <n>]",
		Summary: "Print the log of one CI job, as the runner wrote it.",
		Output:  "The plain log text. The command strips terminal colour codes and nothing else.",
		Flags:   []Flag{{"--tail", "Keep the last <n> lines."}},
		Exits:   []string{"0 printed the log", "1 the call failed, the job is not found, or the job has no log", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "find the first error line, fix the cause, push, then run vcs pr status <n>",
			OutFail:        "a wrapper job has no log. Read the log of the called job, which has a similar name",
			OutUsage:       "give the job-id from vcs ci jobs",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "ci", Name: "rerun", Session: true,
		Usage:   "vcs ci rerun <run-id|run-address> [--failed]",
		Summary: "Restart a CI run. The Forgejo API has no rerun route, so this uses the web route with a session.",
		Output:  "One row: `rerun` and the run-id.",
		Flags:   []Flag{{"--failed", "Restart the failed jobs only."}},
		Exits:   []string{"0 restarted", "1 the call failed, the run is not found, or --failed found no failed job", "64 usage error", "69 no usable driver", "77 no web session, or the server no longer accepts it"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> in a few minutes, or vcs ci runs --limit 1",
			OutFail:        "get the run-id again with vcs ci runs",
			OutUsage:       "give a run-id or the web address of a run",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
			OutNoSession:   "stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat",
		},
	},
	{
		Noun: "attach", Name: "",
		Usage:   "vcs attach <n> <file>... [--comment] [--title <text>]",
		Summary: "Upload files to an issue or a pull request, and optionally post one comment that shows them.",
		Output:  "One row for each file: name, download address. With --comment, a last line `comment posted`.",
		Flags: []Flag{
			{"--comment", "Post one comment that embeds each file as an image."},
			{"--title", "The first line of the comment. The default is Screenshots."},
		},
		Exits: []string{"0 uploaded", "1 an upload or the comment failed", "64 usage error, or a file is not a plain file", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token. Run the command again for the files that did not upload",
			OutUsage:       "give the number, then one or more plain files",
			OutUnavailable: "stop and ask the user. This host has no upload route",
		},
	},
	{
		Noun: "session", Name: "login",
		Usage:   "VCS_PASSWORD=... vcs session login --user <name>",
		Summary: "Sign in to the web side of the forge and keep the session. The password comes from VCS_PASSWORD, never from an argument.",
		Output:  "`ok`.",
		Flags:   []Flag{{"--user", "The user name. VCS_USER is the fallback."}},
		Exits:   []string{"0 signed in", "1 the login was refused, or the account needs a second factor", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "stop and ask the user. For a second factor the user runs vcs session import",
			OutUsage:       "stop and ask the user to run this command. Never put a password in a command",
			OutUnavailable: "",
		},
	},
	{
		Noun: "session", Name: "import",
		Usage:   "vcs session import < cookie-value",
		Summary: "Keep the i_like_gitea cookie of a browser session, read from standard input. Use it for an account with a second factor.",
		Output:  "`ok`.",
		Exits:   []string{"0 saved", "1 the server did not accept the cookie", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "stop and ask the user for a fresh cookie",
			OutUsage:       "give the cookie value on standard input",
			OutUnavailable: "",
		},
	},
	{
		Noun: "session", Name: "status",
		Usage:   "vcs session status",
		Summary: "Tell whether the kept session is still good.",
		Output:  "`ok`.",
		Exits:   []string{"0 the session is good", "64 usage error", "77 no session, or the server no longer accepts it"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "",
			OutUsage:       "run vcs session status with no argument",
			OutUnavailable: "",
			OutNoSession:   "stop and ask the user to run vcs session login, or vcs session import",
		},
		Session: true,
	},
	{
		Noun: "session", Name: "logout",
		Usage:   "vcs session logout",
		Summary: "Delete the kept session of this host.",
		Output:  "`ok`.",
		Exits:   []string{"0 deleted, or there was none", "1 the delete failed", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the permissions of the config directory",
			OutUsage:       "run vcs session logout with no argument",
			OutUnavailable: "",
		},
	},
	{
		Noun: "docs", Name: "gen",
		Usage:   "vcs docs gen <dir>",
		Summary: "Write one Markdown page for each verb of the command tree.",
		Output:  "One line for each page written.",
		Exits:   []string{"0 wrote the pages", "1 a write failed", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "commit the changed pages with the code change",
			OutFail:        "check that the directory exists and is writable",
			OutUsage:       "give one directory, such as docs/Commands",
			OutUnavailable: "",
		},
	},
}

// Key names a verb: "ci runs", or "attach" for a noun with no verb.
func (v Verb) Key() string {
	if v.Name == "" {
		return v.Noun
	}
	return v.Noun + " " + v.Name
}

// Find returns the verb of a noun and a name.
func Find(noun, name string) (Verb, bool) {
	for _, v := range Tree {
		if v.Noun == noun && v.Name == name {
			return v, true
		}
	}
	return Verb{}, false
}
