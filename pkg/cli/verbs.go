package cli

// Exit codes. They follow sysexits.h where it fits.
const (
	ExitOK          = 0
	ExitFail        = 1
	ExitUsage       = 64
	ExitLint        = 65
	ExitUnavailable = 69
	ExitNoSession   = 77
)

// Outcomes that every verb has a hint for.
const (
	OutOK          = "ok"
	OutFail        = "fail"
	OutUsage       = "usage"
	OutLint        = "lint"
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
	// Sends is true when the verb sends text that the prose linters check first.
	Sends bool
	Exits []string
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
		Exits:   []string{"0 printed the row", "1 the call failed or the pull request isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "if ci is failure, run vcs ci runs --sha <head sha> --limit 3, then vcs ci jobs <run-id>",
			OutFail:        "check the number with vcs pr list, then run the command again",
			OutUsage:       "run vcs pr status <n> with the number of one pull request",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "create", Sends: true,
		Usage:   "vcs pr create <title> --head <branch> [--base <branch>] (--body <text> | --body-file <path>) [--draft]",
		Summary: "Open a pull request. The command checks the branch name and the AI credit line, and the prose lint runs on the title and body.",
		Output:  "One row: the number and the web address.",
		Flags: []Flag{
			{"--head", "The branch that holds the work. It must match type/short-description."},
			{"--base", "The branch to merge into. The default is main."},
			{"--body", "The body text."},
			{"--body-file", "A file that holds the body. Use - for standard input."},
			{"--draft", "Open the pull request as a draft."},
		},
		Exits: []string{"0 opened", "1 the call failed", "64 usage error", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "run vcs pr watch <n> until CI ends",
			OutFail:        "check the token, and that you pushed the branch",
			OutUsage:       "give a title, --head, and one of --body or --body-file",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "issue", Name: "create", Sends: true,
		Usage:   "vcs issue create <title> (--body <text> | --body-file <path>)",
		Summary: "Open an issue. The command refuses the AI credit line, and the prose lint runs on the title and body.",
		Output:  "One row: the number and the web address.",
		Flags: []Flag{
			{"--body", "The body text."},
			{"--body-file", "A file that holds the body. Use - for standard input."},
		},
		Exits: []string{"0 opened", "1 the call failed", "64 usage error", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token, then run the command again",
			OutUsage:       "give a title and one of --body or --body-file",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "ready",
		Usage:   "vcs pr ready <n>",
		Summary: "Take a pull request out of draft. A pull request that isn't a draft stays as it is.",
		Output:  "One line: `ready` and the number.",
		Exits:   []string{"0 ready", "1 the call failed or the pull request isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to check the merge state",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "edit", Sends: true,
		Usage:   "vcs pr edit <n> [--title <text>] [--body <text> | --body-file <path>]",
		Summary: "Change the title or the body of a pull request. The command refuses the AI credit line, and the prose lint runs on the new text. A draft stays a draft.",
		Output:  "One line: `edited` and the number.",
		Flags: []Flag{
			{"--title", "The new title."},
			{"--body", "The new body text."},
			{"--body-file", "A file that holds the new body. Use - for standard input."},
		},
		Exits: []string{"0 edited", "1 the call failed or the pull request isn't found", "64 usage error", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver, or the driver has no pr edit"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to check the pull request",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number and at least one of --title, --body or --body-file",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no pr edit yet",
		},
	},
	{
		Noun: "pr", Name: "watch",
		Usage:   "vcs pr watch <n> [--interval <seconds>] [--timeout <seconds>]",
		Summary: "Poll the checks and the merge state of one pull request until both are good.",
		Output:  "One row each time the state changes, in the format of vcs pr status.",
		Flags: []Flag{
			{"--interval", "Seconds between two polls. The default is 30."},
			{"--timeout", "Seconds to wait before the command gives up. The default is 3600."},
		},
		Exits: []string{"0 the pull request merged, or its checks are green and its merge state is clean", "1 a check failed, the pull request has a conflict or closed, the call failed, or the wait timed out", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "the pull request is good. Hand it off, or mark it ready with vcs pr ready <n>",
			OutFail:        "read the last row. If ci is failure, run vcs ci runs --sha <head sha> --limit 3 and vcs ci log. If conflicted files show, merge main in. Then run vcs pr watch <n> again",
			OutUsage:       "run vcs pr watch <n> with the number of one pull request",
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
			{"--status", "Keep one status: failure or success."},
			{"--limit", "Keep the newest <n> rows. The default is 20. 0 reads every run, up to 1000."},
		},
		Exits: []string{"0 printed the rows (an empty list isn't an error)", "1 the call failed", "64 usage error", "69 no usable driver"},
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
		Output:  "One row for each job, tab-separated: job-id, name, status, attempt, note. The note is `no-log` on a wrapper job.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the run isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "for a failed job, run vcs ci log <job-id> --tail 80",
			OutFail:        "get the run-id again with vcs ci runs",
			OutUsage:       "give a run-id or the web address of a run",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "ci", Name: "log",
		Usage:   "vcs ci log <job-id> [--tail <n>]",
		Summary: "Print the log of one CI job, as the runner wrote it.",
		Output:  "The plain log text. The command strips terminal colour codes and nothing else.",
		Flags:   []Flag{{"--tail", "Keep the last <n> lines."}},
		Exits:   []string{"0 printed the log", "1 the call failed, the job isn't found, or the job has no log", "64 usage error", "69 no usable driver"},
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
		Exits:   []string{"0 restarted", "1 the call failed, the run isn't found, or --failed found no failed job", "64 usage error", "69 no usable driver", "77 no web session, or the server no longer accepts it"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> in a few minutes, or vcs ci runs --limit 1",
			OutFail:        "get the run-id again with vcs ci runs",
			OutUsage:       "give a run-id or the web address of a run",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
			OutNoSession:   "stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat",
		},
	},
	{
		Noun: "comment", Name: "add", Sends: true,
		Usage:   "vcs comment add <n> (--body <text> | --body-file <path>)",
		Summary: "Post one comment on an issue or a pull request. The prose lint runs first.",
		Output:  "One line: `comment posted`.",
		Flags: []Flag{
			{"--body", "The comment text."},
			{"--body-file", "A file that holds the comment text. Use - for standard input."},
		},
		Exits: []string{"0 posted", "1 the call failed", "64 usage error, or both or neither of the body flags", "65 the text fails the prose lint, and the command sends nothing", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token and the number, then run the command again",
			OutUsage:       "give the number and one of --body or --body-file",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "board", Name: "list",
		Usage:   "vcs board list [--state open|closed]",
		Summary: "List the project boards of the repository.",
		Output:  "One row for each board: id, title, address.",
		Flags:   []Flag{{"--state", "open or closed. The default is open."}},
		Exits:   []string{"0 listed", "1 the call failed", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token and the host",
			OutUsage:       "use --state open or --state closed",
			OutUnavailable: "stop and ask the user. This host has no board support",
		},
	},
	{
		Noun: "board", Name: "view",
		Usage:   "vcs board view <board-id>",
		Summary: "Show the cards of one board.",
		Output:  "One row for each card: column id, column title, item id, number, title. A column with no cards gives no row.",
		Exits:   []string{"0 shown", "1 the call failed or the board isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "move a card with vcs board move <board-id> <column> <n>",
			OutFail:        "get the board id again with vcs board list",
			OutUsage:       "give one board id",
			OutUnavailable: "stop and ask the user. This host has no board support",
		},
	},
	{
		Noun: "board", Name: "add", Session: true,
		Usage:   "vcs board add <board-id> <n>...",
		Summary: "Put issues or pull requests on a board. Forgejo needs a web session for this.",
		Output:  "One line: `ok`.",
		Exits:   []string{"0 added", "1 the call failed or an issue isn't found", "64 usage error", "69 no usable driver", "77 no web session, or the server no longer accepts it"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the board id and the numbers",
			OutUsage:       "give the board id, then one or more numbers",
			OutUnavailable: "stop and ask the user. This host has no board support",
			OutNoSession:   "stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat",
		},
	},
	{
		Noun: "board", Name: "move", Session: true,
		Usage:   "vcs board move <board-id> <column> <n>...",
		Summary: "Move cards to a column, by column id or title. Forgejo needs a web session for this.",
		Output:  "One line: `ok`.",
		Exits:   []string{"0 moved", "1 the call failed, or the board, column or issue isn't found", "64 usage error", "69 no usable driver", "77 no web session, or the server no longer accepts it"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "read the column ids with vcs board view <board-id>",
			OutUsage:       "give the board id, the column, then one or more numbers",
			OutUnavailable: "stop and ask the user. This host has no board support",
			OutNoSession:   "stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat",
		},
	},
	{
		Noun: "attach", Name: "", Sends: true,
		Usage:   "vcs attach <n> <file>... [--comment] [--title <text>]",
		Summary: "Upload files to an issue or a pull request, and optionally post one comment that shows them.",
		Output:  "One row for each file: name, download address. With --comment, a last line `comment posted`.",
		Flags: []Flag{
			{"--comment", "Post one comment that embeds each file as an image."},
			{"--title", "The first line of the comment. The default is Screenshots."},
		},
		Exits: []string{"0 uploaded", "1 an upload or the comment failed", "64 usage error, or a file isn't a plain file", "65 the comment text fails the prose lint, and the command sends nothing", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token. Run the command again for the files that didn't upload",
			OutUsage:       "give the number, then one or more plain files",
			OutLint:        "fix each finding that vcs printed in the comment text or the --title, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no upload route",
		},
	},
	{
		Noun: "session", Name: "login",
		Usage:   "VCS_PASSWORD=... vcs session login --user <name>",
		Summary: "Sign in to the web side of the forge and keep the session. The password comes from VCS_PASSWORD, never from an argument.",
		Output:  "`ok`.",
		Flags:   []Flag{{"--user", "The user name. VCS_USER is the fallback."}},
		Exits:   []string{"0 signed in", "1 the server refused the login, or the account needs a second factor", "64 usage error"},
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
		Exits:   []string{"0 saved", "1 the server didn't accept the cookie", "64 usage error"},
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
