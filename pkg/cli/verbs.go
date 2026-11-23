package cli

// Exit codes. They follow sysexits.h where it fits.
const (
	ExitOK          = 0
	ExitFail        = 1
	ExitUsage       = 64
	ExitUnavailable = 69
)

// Outcomes that every verb has a hint for.
const (
	OutOK          = "ok"
	OutFail        = "fail"
	OutUsage       = "usage"
	OutUnavailable = "unavailable"
)

// Verb is one command of the command tree. The docs build from this table.
type Verb struct {
	Noun    string
	Name    string
	Usage   string
	Summary string
	Output  string
	Flags   []Flag
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
		Exits:   []string{"0 printed the row", "1 the call failed or the pull request isn't found", "64 usage error", "69 no usable driver"},
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

// Find returns the verb of a noun and a name.
func Find(noun, name string) (Verb, bool) {
	for _, v := range Tree {
		if v.Noun == noun && v.Name == name {
			return v, true
		}
	}
	return Verb{}, false
}
