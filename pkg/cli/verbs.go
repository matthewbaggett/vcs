package cli

import "git.grey.ooo/packages/vcs/pkg/session"

// Exit codes. They follow sysexits.h where it fits.
const (
	ExitOK          = 0
	ExitFail        = 1
	ExitFound       = 2 // stopwatch list --check found a stopwatch. It means "found", not "failed".
	ExitConflict    = 2 // pr watch: the pull request conflicts with its base branch
	ExitClosed      = 3 // pr watch: the pull request is closed or merged
	ExitTimeout     = 4 // pr watch: the timeout came first
	ExitBusy        = 5 // pr watch: another watcher of this pull request is alive
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
	OutClosed      = "closed"
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
		Usage:   "vcs pr status <n> [--pushed]",
		Summary: "Print the state of one pull request on one line.",
		Output:  "One row, tab-separated: state, draft, mergeable, conflicted files, ci, head sha, url. With --pushed, an eighth column holds the commit time of the head commit, as Zulu time.",
		Flags:   []Flag{{"--pushed", "Add the commit time of the head commit as the last column. It costs one more request."}},
		Exits:   []string{"0 printed the row", "1 the call failed or the pull request isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "if ci is failure, run vcs ci runs --sha <head sha> --limit 3, then vcs ci jobs <run-id>",
			OutFail:        "check the number with vcs pr list, then run the command again",
			OutUsage:       "run vcs pr status <n> with the number of one pull request",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "statuses",
		Usage:   "vcs pr statuses <sha> [--latest] [--description]",
		Summary: "Print the status rows of one commit, one line for each row.",
		Output:  "Rows, tab-separated, no header: context, state, address. The server lists every row it ever got, newest first. A state is success, pending, failure, error, or warning. With --description, a fourth column holds the description of the check. An empty list isn't an error.",
		Flags: []Flag{
			{"--latest", "Keep the newest row of each context, sorted by context."},
			{"--description", "Add the description as the last column."},
		},
		Exits: []string{"0 printed the rows", "1 the call failed or the commit isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "if a state is failure, run vcs ci runs --sha <sha> --limit 3, then vcs ci jobs <run-id>",
			OutFail:        "check the commit with git log, then run the command again",
			OutUsage:       "run vcs pr statuses <sha> with one commit",
			OutUnavailable: "stop and ask the user. This host has no driver for statuses yet",
		},
	},
	{
		Noun: "issue", Name: "timeline",
		Usage:   "vcs issue timeline <n>",
		Summary: "Print the events of one issue or pull request, oldest first.",
		Output:  "Rows, tab-separated, no header: event id, created, type, user, detail. The type is the one that the server gives, such as comment, label, change_title, pull_push, or commit_ref. The detail of a label is +name or -name. The detail of a title change is old -> new. The detail of a push is push or force-push with the commit ids. Other types print their text on one line. An empty timeline isn't an error.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the issue isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "compare the time of the last event with the time of your claim, then decide if the claim is stale",
			OutFail:        "check the number with vcs issue list, then run the command again",
			OutUsage:       "run vcs issue timeline <n> with the number of one issue or pull request",
			OutUnavailable: "stop and ask the user. This host has no driver for timelines yet",
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
			{"--trace", traceHelp},
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
			{"--trace", traceHelp},
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
		Exits:   []string{"0 ready", "1 the call failed, the pull request isn't found, or the pull request has the state closed or merged", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to check the merge state",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "draft",
		Usage:   "vcs pr draft <n>",
		Summary: "Turn a pull request back into a draft. A draft stays a draft.",
		Output:  "One line: `draft` and the number.",
		Exits:   []string{"0 draft", "1 the call failed, the pull request isn't found, or the pull request has the state closed or merged", "64 usage error", "69 no usable driver, or the driver has no draft support"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to check the draft flag",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "close",
		Usage:   "vcs pr close <n>",
		Summary: "Close a pull request with no merge.",
		Output:  "One line: `closed` and the number.",
		Exits:   []string{"0 closed", "1 the call failed or the pull request isn't found", "64 usage error", "69 no usable driver, or the driver has no close support"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "pr", Name: "rebase",
		Usage:   "vcs pr rebase <n>",
		Summary: "Tick the rebase box in the body of a Renovate pull request. Renovate then rebases the branch. Forgejo only.",
		Output:  "One line: `ok` when the command ticks the box, or `already` when the box has a tick.",
		Exits:   []string{"0 ticked, or already ticked", "1 the call failed, or the pull request has no rebase box", "64 usage error", "69 no usable driver, or the host has no rebase box"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "only a Renovate pull request has a rebase box. Check the number",
			OutUsage:       "give the pull request number",
			OutUnavailable: "stop and ask the user. This host has no rebase box",
		},
	},
	{
		Noun: "pr", Name: "edit", Sends: true,
		Usage:   "vcs pr edit <n> [--title <text>] [--body <text> | --body-file <path>] [--base <branch>]",
		Summary: "Change the title, the body or the base branch of a pull request. The command refuses the AI credit line, and the prose lint runs on the new text. A draft stays a draft.",
		Output:  "One line: `edited` and the number.",
		Flags: []Flag{
			{"--title", "The new title."},
			{"--body", "The new body text."},
			{"--body-file", "A file that holds the new body. Use - for standard input."},
			{"--base", "The new base branch."},
			{"--trace", traceHelp},
		},
		Exits: []string{"0 edited", "1 the call failed or the pull request isn't found", "64 usage error", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver, or the driver has no pr edit"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to check the pull request",
			OutFail:        "check the number and the token",
			OutUsage:       "give the pull request number and at least one of --title, --body, --body-file or --base",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no pr edit yet",
		},
	},
	{
		Noun: "pr", Name: "merge", Sends: true,
		Usage:   "vcs pr merge <n> [--method merge|rebase|rebase-merge|squash] [--delete-branch]",
		Summary: "Merge a pull request. Run it only when the owner has said to merge that pull request. The command doesn't check the checks, so run vcs pr status first.",
		Output:  "One line: `merged` and the number.",
		Flags: []Flag{
			{"--method", "How to merge. The default is merge."},
			{"--delete-branch", "Delete the head branch after the merge."},
		},
		Exits: []string{"0 merged", "1 the call failed, the server refused the merge, or the pull request already merged", "64 usage error", "69 no usable driver, or the driver has no pr merge"},
		Hints: map[string]string{
			OutOK:          "run vcs pr status <n> to confirm the state says merged",
			OutFail:        "read the message. Check the merge state with vcs pr status <n>",
			OutUsage:       "give the pull request number. The method is merge, rebase, rebase-merge or squash",
			OutLint:        "the command sends no text, so this exit doesn't occur. Check the arguments",
			OutUnavailable: "stop and ask the user. This host has no pr merge yet",
		},
	},
	{
		Noun: "pr", Name: "heal",
		Usage:   "vcs pr heal ensure | add <n> | remove <n> | sweep --live <n,n,...|none>",
		Summary: "Mark and unmark the pull requests that a worker heals now, with the label `➕healing`. The environment variable VCS_HEAL_LABEL changes the name. The sweep clears the mark of each pull request that closed, has no live worker, or is healthy.",
		Output:  "`ensure` prints `created` or `exists`. `add` prints `ok`. `remove` prints `ok`, or `absent` when the pull request has no mark. `sweep` prints one row for each mark it cleared: `cleared`, the number, and the reason (`closed`, `no-worker`, or `healthy`).",
		Flags: []Flag{
			{"--live", "For sweep: the numbers of the pull requests that a worker holds now, separated by commas, or `none`."},
		},
		Exits: []string{"0 done", "1 a call failed", "64 usage error", "69 no usable driver, or the driver has no label, issue edit, or pr list"},
		Hints: map[string]string{
			OutOK:          "the mark is in the state you asked for",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "use `ensure`, `add <n>`, `remove <n>`, or `sweep --live <numbers>`. Separate the numbers with commas, or give `none`",
			OutUnavailable: "stop and ask the user. This host has no label, issue edit, or pr list yet",
		},
	},
	{
		Noun: "pr", Name: "tick", Sends: false,
		Usage:   "vcs pr tick [--dry-run]",
		Summary: "Read every open pull request once. Merge each one that isn't a draft, merges clean, targets main, and has green checks. Report each one that needs a person. Run it only when the owner has said to merge ready pull requests.",
		Output:  "One row for each pull request that needs attention: a word, the number, the head sha. The words: `MERGED`, `MERGED-BUT-NOT-LANDED`, `MERGE-FAILED`, `RED`, `NOT-MERGEABLE` (with ci=<state>), `DRAFT-GREEN`, `DRAFT-RED`, and `WOULD-MERGE`. With nothing to report, it prints `QUIET`.",
		Flags: []Flag{
			{"--dry-run", "Merge nothing. Print `WOULD-MERGE` for each pull request that's ready."},
		},
		Exits: []string{"0 the pass ran. The rows hold the result", "1 the call to list the pull requests failed", "64 usage error", "69 no usable driver, or the driver has no pr list or pr merge"},
		Hints: map[string]string{
			OutOK:          "act on each row: fix a `RED` or `NOT-MERGEABLE` pull request, mark a `DRAFT-GREEN` one ready, and check a `MERGE-FAILED` one with vcs pr status <n>",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "run vcs pr tick with no argument, or with --dry-run",
			OutUnavailable: "stop and ask the user. This host has no pr list or pr merge yet",
		},
	},
	{
		Noun: "pr", Name: "watch",
		Usage:   "vcs pr watch <n> [--interval <seconds>] [--timeout <seconds>] [--remote <name>]\n       vcs pr watch <n> --status",
		Summary: "Poll the checks and the merge state of one pull request until both settle.",
		Output:  "One line each time the state changes: time, number, head, ci, and merge. Then a RESULT line. A failed check or a conflict adds a list before the RESULT line. With --status: watching #<n> pid=<pid>, or not watching #<n>.",
		Flags: []Flag{
			{"--interval", "Seconds between two polls. The default is 30."},
			{"--timeout", "Seconds to wait before the command gives up. The default is 3600."},
			{"--remote", "The git remote that holds the pull request refs. The default is origin."},
			{"--status", "Tell whether a watcher of this pull request is alive. Exit 0 if one is, 1 if none is."},
		},
		Exits: []string{"0 every check passed, every required check has a status, and the merge is clean", "1 a check failed, or a read failed", "2 the pull request conflicts with its base branch", "3 the pull request closed or merged", "4 the timeout came first", "5 another watcher of this pull request is alive", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "the pull request is good. Hand it off, or mark it ready with vcs pr ready <n>",
			OutFail:        "read the last lines. If a check failed, run vcs ci runs --sha <head sha> --limit 3 and vcs ci log. If conflicted files show, merge main in. Then run vcs pr watch <n> again",
			OutClosed:      "the pull request has landed or someone closed it. You can't watch it or mark it ready. Run vcs pr status <n> to see which. If it landed, pull the base branch",
			OutUsage:       "run vcs pr watch <n> with the number of one pull request",
			OutUnavailable: "stop and ask the user. This host has no driver yet",
		},
	},
	{
		Noun: "package", Name: "list",
		Usage:   "vcs package list [<name>] [--stale-days <n>]",
		Summary: "List the generic packages of the owner of the repository, and report the old versions. The verb never deletes. No other registry type is reachable.",
		Output:  "One row for each version, newest first within a name: name, version, created, age in days, status. The status is newest, old (at least --stale-days old, default 30), or recent. A name gets a note on stderr when old versions exist.",
		Flags:   []Flag{{"--stale-days <n>", "The age in days from which an older version counts as old. The default is 30."}},
		Exits:   []string{"0 listed, or no package exists", "1 the call failed", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the host, the owner, and the token",
			OutUsage:       "give at most one package name of lower-case letters, digits, dots, underscores, and dashes",
			OutUnavailable: "stop and ask the user. This host has no package verbs yet",
			OutNoSession:   "stop and ask the user for a token with the read:package scope",
		},
	},
	{
		Noun: "package", Name: "upload",
		Usage:   "vcs package upload <name> <version> <file> [--sha256 <hex>] [--max-mib <n>] [--stale-days <n>] [--dry-run]",
		Summary: "Upload one file as a version of a generic package, and link the package to the repository. An upload never replaces: an existing version stops the verb. The file must be a plain file, not a link, not empty, and within the size limit.",
		Output:  "A first row: ok, the download address, the checksum of the file. With --dry-run the first word is dry-run, and the verb sends nothing. Then the list rows of the same name, so the old versions show at once.",
		Flags: []Flag{
			{"--sha256 <hex>", "Stop with exit 1 unless the file has this checksum."},
			{"--max-mib <n>", "The size limit in MiB. The default is 512."},
			{"--stale-days <n>", "The age in days from which an older version counts as old. The default is 30."},
			{"--dry-run", "Check everything and print the address and the checksum. Send nothing."},
		},
		Exits: []string{"0 uploaded, or the dry run passed", "1 a call failed, the version exists, or the checksum differs", "64 usage error, or a name, a version, or a file that the scope rules refuse", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "read the message. An existing version needs a new version number, or a delete on purpose",
			OutUsage:       "give a name, a version, and a plain file",
			OutUnavailable: "stop and ask the user. This host has no package verbs yet",
			OutNoSession:   "stop and ask the user for a token with the write:package scope",
		},
	},
	{
		Noun: "package", Name: "delete",
		Usage:   "vcs package delete <name> <version> [--allow-newest]",
		Summary: "Delete one version of a generic package. The verb refuses the newest version unless you pass the flag for it.",
		Output:  "ok.",
		Flags:   []Flag{{"--allow-newest", "Delete the newest version on purpose."}},
		Exits:   []string{"0 deleted", "1 a call failed, the version is missing, or it's the newest", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "list the versions with vcs package list <name>",
			OutUsage:       "give a name and a version",
			OutUnavailable: "stop and ask the user. This host has no package verbs yet",
			OutNoSession:   "stop and ask the user for a token with the write:package scope",
		},
	},
	{
		Noun: "pr", Name: "bind",
		Usage:   "vcs pr bind [<number>] [--branch <name>]",
		Summary: "Find the pull request of a branch and print what a session needs to start work on it: the state, the CI word, and each red job. It only reads. It starts no watcher. With two open pull requests on one branch, it takes the newest.",
		Output:  "A pr row: pr, number, state, draft, mergeable, conflicted files, ci, head sha, url. Then one red row for each red job: red, run id, job id, job name, first error. A line on stderr says how to wait: vcs pr watch.",
		Flags:   []Flag{{"--branch <name>", "The head branch to search for. The default is the current branch of the checkout."}},
		Exits:   []string{"0 printed the rows", "1 no open pull request has the branch, or a call failed", "64 usage error", "69 no usable driver, or the driver has no pr list"},
		Hints: map[string]string{
			OutOK:          "read each red row with vcs ci log <job id>, or wait with vcs pr watch <number>",
			OutFail:        "check the branch name, and that the pull request is open",
			OutUsage:       "give one number, or a --branch, not both",
			OutUnavailable: "stop and ask the user. This host has no pr list yet",
		},
	},
	{
		Noun: "pr", Name: "list",
		Usage:   "vcs pr list [--state open|closed|all] [--label <name,name>] [--milestone <title>] [--ci] [--limit <n>]",
		Summary: "List pull requests, newest first.",
		Output:  "One row for each pull request, tab-separated: number, title, head branch, base branch, mergeable, ci, labels, milestone. The ci column is empty unless you pass --ci. Labels are comma-separated.",
		Flags: []Flag{
			{"--state", "Keep open, closed, or all pull requests. The default is open."},
			{"--label", "Keep the pull requests that carry every label. Separate names with commas."},
			{"--milestone", "Keep the pull requests of one milestone."},
			{"--ci", "Fill the ci column. It costs one more request for each pull request."},
			{"--limit", "Keep the newest <n> rows after the filter. The default keeps all."},
		},
		Exits: []string{"0 printed the rows (an empty list isn't an error)", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no pr list"},
		Hints: map[string]string{
			OutOK:          "take a number from the first column and run vcs pr status <n>",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "see vcs pr list for the flags",
			OutUnavailable: "stop and ask the user. This host has no pr list yet",
		},
	},
	{
		Noun: "issue", Name: "list",
		Usage:   "vcs issue list [--state open|closed|all] [--label <name,name>] [--milestone <title>] [--query <text>] [--limit <n>]",
		Summary: "List issues, newest first. The list holds no pull request.",
		Output:  "One row for each issue, tab-separated: number, title, state, labels, milestone. Labels are comma-separated.",
		Flags: []Flag{
			{"--state", "Keep open, closed, or all issues. The default is open."},
			{"--label", "Keep the issues that carry every label. Separate names with commas."},
			{"--milestone", "Keep the issues of one milestone."},
			{"--query", "Keep the issues whose title or body matches the text. The server filters."},
			{"--limit", "Keep the newest <n> rows after the filter. The default keeps all."},
		},
		Exits: []string{"0 printed the rows (an empty list isn't an error)", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no issue list"},
		Hints: map[string]string{
			OutOK:          "take a number from the first column and run vcs issue view <n>",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "see vcs issue list for the flags",
			OutUnavailable: "stop and ask the user. This host has no issue list yet",
		},
	},
	{
		Noun: "issue", Name: "newer",
		Usage:   "vcs issue newer <index> [--state open|all] [--kind issue|any]",
		Summary: "List the issues with an index higher than a mark, newest first.",
		Output:  "One row for each item, tab-separated, no header: index, kind, user, title. The kind is issue or pr. The user is the login that opened the item. Issues and pull requests share one index space, so a mark moves on a pull request. An empty list isn't an error.",
		Flags: []Flag{
			{"--state", "Keep open items, or all. The default is open."},
			{"--kind", "Keep issues only, or any item with a pull request too. The default is issue."},
		},
		Exits: []string{"0 printed the rows", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no index list"},
		Hints: map[string]string{
			OutOK:          "take an index from the first column and run vcs issue view <n>, or keep the highest index as the new mark",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "run vcs issue newer <index> with a whole number, 0 or more",
			OutUnavailable: "stop and ask the user. This host has no index list yet",
		},
	},
	{
		Noun: "issue", Name: "top",
		Usage:   "vcs issue top",
		Summary: "Print the highest index of any issue or pull request.",
		Output:  "One number: the highest index in any state, or 0 when the repository has none.",
		Exits:   []string{"0 printed the number", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no index list"},
		Hints: map[string]string{
			OutOK:          "keep the number as the mark, then run vcs issue newer <mark> in the next round",
			OutFail:        "run the command again. If it fails twice, stop and report the error",
			OutUsage:       "run vcs issue top with no argument",
			OutUnavailable: "stop and ask the user. This host has no index list yet",
		},
	},
	{
		Noun: "issue", Name: "view",
		Usage:   "vcs issue view <n>",
		Summary: "Print one issue with its body and its comments.",
		Output:  "A first row, tab-separated: number, title, state, labels, milestone, global id, web address. The id and the address stay empty when the driver has none. Then a blank line and the body. Then each comment: a blank line, a row of `comment`, the writer, and the date, and the text.",
		Exits:   []string{"0 printed the issue", "1 the call failed or the issue isn't found", "64 usage error", "69 no usable driver, or the driver has no issue view"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the number with vcs issue list",
			OutUsage:       "run vcs issue view <n> with the number of one issue",
			OutUnavailable: "stop and ask the user. This host has no issue view yet",
		},
	},
	{
		Noun: "ci", Name: "failed",
		Usage:   "vcs ci failed <pr-number|sha>",
		Summary: "List the red jobs of a pull request head or a commit, with the first error of each log.",
		Output:  "One row for each red job, tab-separated: run-id, job-id, job-name, first error. The first error is the text of the first `::error::` line. The column is blank when the log has none, and `(log unreadable)` when the read failed. A wrapper job has no log and a blank column.",
		Exits:   []string{"0 printed the rows (a commit with no red job prints nothing)", "1 the call failed, the target isn't found, or a run belongs to another commit", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "read the first-error column before you call a failure a flake. For the whole log, run vcs ci log <job-id> --tail 80",
			OutFail:        "check the number or the sha, then run the command again",
			OutUsage:       "give a pull request number (1 to 6 digits) or a commit sha (7 to 40 hex characters)",
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
		Output:  "One row for each job, tab-separated: job-id, name, status, attempt, note. The note is `no-log` on a wrapper job. When the run made artifacts, one row follows for each artifact: `artifact`, artifact-id, name, uncompressed size in bytes, note. The note is `expired` for an artifact that the server removed. The server ties an artifact to the run, not to a job, so these rows come after all the job rows.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the run isn't found", "64 usage error", "69 no usable driver"},
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
		Output:  "One row for each artifact, tab-separated: artifact-id, name, uncompressed size in bytes, note. The note is `expired` for an artifact that the server removed. An empty list isn't an error.",
		Exits:   []string{"0 printed the rows", "1 the call failed or the run isn't found", "64 usage error", "69 no usable driver, or the host has no artifact route"},
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
		Summary: "Download one artifact of a CI run, or every artifact with --all. The server sends each artifact as a zip archive, and the command writes it as it is. It doesn't unpack it.",
		Output:  "One row for each file written, tab-separated: path, uncompressed size in bytes.",
		Flags: []Flag{
			{"--all", "Download every artifact that hasn't expired. Give no name."},
			{"-o", "The file to write. The default is `<name>.zip` in the current directory, and the command refuses to replace a file that exists. With --all, the path is an existing directory that takes each `<name>.zip`."},
		},
		Exits: []string{"0 wrote the files", "1 the call failed, the run or the artifact isn't found, the artifact expired, or a default file exists", "64 usage error", "69 no usable driver, or the host has no artifact route"},
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
		Output:  "One row: the comment id and its web address.",
		Flags: []Flag{
			{"--body", "The comment text."},
			{"--body-file", "A file that holds the comment text. Use - for standard input."},
			{"--trace", traceHelp},
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
		Noun: "comment", Name: "list",
		Usage:   "vcs comment list <n>",
		Summary: "List the comments of an issue or a pull request.",
		Output:  "One row for each comment, tab-separated: id, user, created, body, updated. The body is on one line, and each newline is the two characters \\n.",
		Exits:   []string{"0 listed", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no comment support"},
		Hints: map[string]string{
			OutOK:          "use the id with vcs comment edit or vcs comment delete",
			OutFail:        "check the token and the number",
			OutUsage:       "give the number of an issue or a pull request",
			OutUnavailable: "stop and ask the user. This host has no comment support yet",
		},
	},
	{
		Noun: "comment", Name: "edit", Sends: true,
		Usage:   "vcs comment edit <id> (--body <text> | --body-file <path>)",
		Summary: "Replace the text of one comment. Find the id with vcs comment list. The prose lint runs first.",
		Output:  "One line: `ok`.",
		Flags: []Flag{
			{"--body", "The new comment text."},
			{"--body-file", "A file that holds the new comment text. Use - for standard input."},
			{"--trace", traceHelp},
		},
		Exits: []string{"0 edited", "1 the call failed or the comment isn't found", "64 usage error, or both or neither of the body flags", "65 the text fails the prose lint, and the command sends nothing", "69 no usable driver, or the driver has no comment support"},
		Hints: map[string]string{
			OutOK:          "run vcs comment list to check the text",
			OutFail:        "get the id again with vcs comment list",
			OutUsage:       "give the comment id and one of --body or --body-file",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no comment support yet",
		},
	},
	{
		Noun: "comment", Name: "delete",
		Usage:   "vcs comment delete <id>",
		Summary: "Delete one comment. Find the id with vcs comment list. A deleted comment can't come back.",
		Output:  "One line: `ok`.",
		Exits:   []string{"0 deleted", "1 the call failed or the comment isn't found", "64 usage error", "69 no usable driver, or the driver has no comment support"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "get the id again with vcs comment list",
			OutUsage:       "give the numeric comment id",
			OutUnavailable: "stop and ask the user. This host has no comment support yet",
		},
	},
	{
		Noun: "milestone", Name: "list",
		Usage:   "vcs milestone list [--state open|closed|all]",
		Summary: "List the milestones of the repository.",
		Output:  "One row for each milestone, tab-separated: id, title, state, open issues, closed issues, due date. The due date looks like 2026-12-01, or the field is empty.",
		Flags:   []Flag{{"--state", "open, closed, or all. The default is open."}},
		Exits:   []string{"0 listed", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no milestone support"},
		Hints: map[string]string{
			OutOK:          "use the title with vcs issue edit --milestone",
			OutFail:        "check the token and the host",
			OutUsage:       "use --state open, closed, or all",
			OutUnavailable: "stop and ask the user. This host has no milestone support yet",
		},
	},
	{
		Noun: "milestone", Name: "view",
		Usage:   "vcs milestone view <id|title>",
		Summary: "Show one milestone. Give the id or the title. A closed milestone counts too.",
		Output:  "One row: id, title, state, open issues, closed issues, due date, description. The description fits on one line, and each newline shows as the two characters \\n.",
		Exits:   []string{"0 shown", "1 the call failed or the milestone isn't found", "64 usage error", "69 no usable driver, or the driver has no milestone support"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "run vcs milestone list --state all to see the titles",
			OutUsage:       "give an id or a one-line title",
			OutUnavailable: "stop and ask the user. This host has no milestone support yet",
		},
	},
	{
		Noun: "milestone", Name: "create", Sends: true,
		Usage:   "vcs milestone create <title> [--due <date>] [--desc <text>]",
		Summary: "Make one milestone. The command refuses the AI credit line, and the prose lint runs on the title and the description first.",
		Output:  "One row: `created` and the title.",
		Flags: []Flag{
			{"--due", "The due date, like 2027-01-31."},
			{"--desc", "A description of the milestone."},
		},
		Exits: []string{"0 created", "1 the call failed, or a milestone with that title exists", "64 usage error, or a bad date", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver, or the driver has no milestone support"},
		Hints: map[string]string{
			OutOK:          "run vcs milestone view <title> to check it",
			OutFail:        "run vcs milestone list --state all to see if the title exists",
			OutUsage:       "give a one-line title, and a date like 2027-01-31 if you use --due",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no milestone support yet",
		},
	},
	{
		Noun: "issue", Name: "edit", Sends: true,
		Usage:   "vcs issue edit <n> [--title <text>] [--body <text> | --body-file <path>] [--state open|closed] [--milestone <title|id|none>] [--add-label <a,b>] [--rm-label <a,b>]",
		Summary: "Change the title, the body, the state, the milestone, or the labels of an issue. The command refuses the AI credit line, and the prose lint runs on the new text. An unknown label or milestone stops the command before anything changes.",
		Output:  "One line: `edited` and the number.",
		Flags: []Flag{
			{"--title", "The new title."},
			{"--body", "The new body text."},
			{"--body-file", "A file that holds the new body. Use - for standard input."},
			{"--trace", traceHelp},
			{"--state", "open to reopen the issue, or closed to close it."},
			{"--milestone", "A milestone title or id. Use none to clear it."},
			{"--add-label", "Labels to add, separated by commas."},
			{"--rm-label", "Labels to remove, separated by commas."},
		},
		Exits: []string{"0 edited", "1 the call failed, or the issue, a label or the milestone isn't found", "64 usage error", "65 a guard rule or the prose lint refuses the text, and the command sends nothing", "69 no usable driver, or the driver has no issue edit"},
		Hints: map[string]string{
			OutOK:          "run vcs issue view <n> to check the issue",
			OutFail:        "check the number, the label names and the token",
			OutUsage:       "give the issue number and at least one change. The state is open or closed",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no issue edit yet",
		},
	},
	{
		Noun: "label", Name: "list",
		Usage:   "vcs label list",
		Summary: "List the labels of the repository.",
		Output:  "One row for each label, tab-separated: id, name, color, description.",
		Exits:   []string{"0 listed", "1 the call failed", "64 usage error", "69 no usable driver, or the driver has no label support"},
		Hints: map[string]string{
			OutOK:          "use the id with vcs label rename",
			OutFail:        "check the token and the host",
			OutUsage:       "give no argument",
			OutUnavailable: "stop and ask the user. This host has no label support yet",
		},
	},
	{
		Noun: "label", Name: "create", Sends: true,
		Usage:   "vcs label create <name> <color> [--desc <text>]",
		Summary: "Make one label. The color is six hex digits, with or without a leading #. The command lints the description, and it leaves the name alone because a name is a tag.",
		Output:  "One row: `created` and the name.",
		Flags:   []Flag{{"--desc", "A one-line description of the label."}},
		Exits:   []string{"0 created", "1 the call failed, or the label exists", "64 usage error, a bad color, or a name or description of more than one line", "65 a guard rule or the prose lint refuses the description, and the command sends nothing", "69 no usable driver, or the driver has no label support"},
		Hints: map[string]string{
			OutOK:          "run vcs label list to check the label",
			OutFail:        "run vcs label list to see if the label exists",
			OutUsage:       "give a one-line name and a color of six hex digits",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This host has no label support yet",
		},
	},
	{
		Noun: "label", Name: "rename",
		Usage:   "vcs label rename <id> <name>",
		Summary: "Change the name of one label. Find the id with vcs label list.",
		Output:  "One row: `renamed`, the id, and the new name.",
		Exits:   []string{"0 renamed", "1 the call failed or the label isn't found", "64 usage error", "69 no usable driver, or the driver has no label support"},
		Hints: map[string]string{
			OutOK:          "run vcs label list to check the name",
			OutFail:        "get the id again with vcs label list",
			OutUsage:       "give a numeric id and a one-line name",
			OutUnavailable: "stop and ask the user. This host has no label support yet",
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
		Noun: "time", Name: "add",
		Usage:   "vcs time add <n> <duration>",
		Summary: "Log time on an issue or a pull request. The duration is like 1h25m, 90m, or 45s.",
		Output:  "One row: `ok`, the new total in seconds, and the new total as 1h25m0s. If vcs can't read the total, one line `ok`.",
		Exits:   []string{"0 logged", "1 the call failed or the issue isn't found", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "vcs logged the time once. Don't run the command again to check it",
			OutFail:        "run vcs time list <n> to see if the entry exists, before you run the command again",
			OutUsage:       "give the number, then a duration of hours, minutes and seconds, like 1h30m",
			OutUnavailable: "stop and ask the user. This host has no time tracking yet",
		},
	},
	{
		Noun: "time", Name: "list",
		Usage:   "vcs time list [<n>] [--mine]",
		Summary: "List the time entries of one issue, or of the whole repository with no number.",
		Output:  "One row for each entry: id, user, seconds, created, issue number.",
		Flags:   []Flag{{"--mine", "Only the entries of the caller."}},
		Exits:   []string{"0 listed", "1 the call failed", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the number and the token",
			OutUsage:       "give no number, or one issue number",
			OutUnavailable: "stop and ask the user. This host has no time tracking yet",
		},
	},
	{
		Noun: "time", Name: "total",
		Usage:   "vcs time total [<n>] [--mine]",
		Summary: "Add up the time entries of one issue, or of the whole repository with no number.",
		Output:  "One row: seconds, then the total as 1h25m0s.",
		Flags:   []Flag{{"--mine", "Only the entries of the caller."}},
		Exits:   []string{"0 printed", "1 the call failed", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the number and the token",
			OutUsage:       "give no number, or one issue number",
			OutUnavailable: "stop and ask the user. This host has no time tracking yet",
		},
	},
	{
		Noun: "workflow", Name: "dispatch",
		Usage:   "vcs workflow dispatch <file> --ref <branch|tag>",
		Summary: "Start one run of a workflow that has a workflow_dispatch trigger. The server gives no run id. Find the run with vcs ci runs. The caller decides whether a manual run is safe.",
		Output:  "ok.",
		Flags:   []Flag{{"--ref <branch|tag>", "The branch or tag to run the workflow at. Required."}},
		Exits:   []string{"0 the server accepted the dispatch", "1 the call failed, or the workflow or the ref is missing", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "find the run with vcs ci runs",
			OutFail:        "check the file name, the ref, and that the workflow has a workflow_dispatch trigger",
			OutUsage:       "give a file name such as nightly.yml and a --ref of a branch or tag",
			OutUnavailable: "stop and ask the user. This host has no workflow dispatch yet",
			OutNoSession:   "stop and ask the user for a token with the write:repository scope",
		},
	},
	{
		Noun: "protection", Name: "list",
		Usage:   "vcs protection list [<branch>]",
		Summary: "List the branch protection rules of the repository. With a branch, keep the rules that cover it. A merge that the server refuses with a green run points at this list.",
		Output:  "One row for each rule: rule, required contexts (a comma list), required approvals, push (none, whitelist, or all), admins (yes or no). No row means the branch has no protection.",
		Exits:   []string{"0 listed, or no rule covers the branch", "1 the call failed", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the host, the repository, and the token",
			OutUsage:       "give at most one branch name",
			OutUnavailable: "stop and ask the user. This host has no protection list yet",
			OutNoSession:   "stop and ask the user for a token. The list needs one, even on a public repository",
		},
	},
	{
		Noun: "runners", Name: "list",
		Usage:   "vcs runners list",
		Summary: "List the CI runners of the whole instance. The call needs a token of an admin user. Match the runner uuid of a failed job log to a name here.",
		Output:  "One row for each runner, sorted by id: id, name, status, labels, uuid, version. The labels are a comma list.",
		Exits:   []string{"0 listed, or no runner exists", "1 the call failed", "64 usage error", "69 no usable driver", "77 the server refused the token: it's missing, too weak, or not an admin"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the host and the token",
			OutUsage:       "this verb takes no argument",
			OutUnavailable: "stop and ask the user. This host has no runner list yet",
			OutNoSession:   "stop and ask an admin for the list, or for a token with the read:admin scope",
		},
	},
	{
		Noun: "secret", Name: "list",
		Usage:   "vcs secret list",
		Summary: "List the Actions secrets of the repository. The server never returns a value.",
		Output:  "One row for each secret, sorted by name: name, created.",
		Exits:   []string{"0 listed, or none exists", "1 the call failed", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the host, the repository, and the token",
			OutUsage:       "this verb takes no argument",
			OutUnavailable: "stop and ask the user. This host has no secret verbs yet",
			OutNoSession:   "stop and ask the user for a token with the write:repository scope",
		},
	},
	{
		Noun: "secret", Name: "set",
		Usage:   "vcs secret set <name> (--value-file <file|-> | --from-env <VAR>)",
		Summary: "Create or replace one Actions secret. The value never comes from an argument. Read it from a file, from standard input with -, or from an environment variable. The verb cuts the trailing newlines and prints no value.",
		Output:  "ok.",
		Flags: []Flag{
			{"--value-file <file|->", "Read the value from a file, or from standard input for -."},
			{"--from-env <VAR>", "Read the value from this environment variable."},
		},
		Exits: []string{"0 set", "1 the call failed", "64 usage error, a bad name, or an empty value. The verb sends nothing", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the host, the repository, and the token",
			OutUsage:       "give a name of letters, digits and underscores, and exactly one of --value-file or --from-env. The value must not be empty",
			OutUnavailable: "stop and ask the user. This host has no secret verbs yet",
			OutNoSession:   "stop and ask the user for a token with the write:repository scope",
		},
	},
	{
		Noun: "secret", Name: "delete",
		Usage:   "vcs secret delete <name>",
		Summary: "Delete one Actions secret.",
		Output:  "ok.",
		Exits:   []string{"0 deleted", "1 the call failed, or the secret doesn't exist", "64 usage error", "69 no usable driver", "77 the server refused the token"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the name with vcs secret list",
			OutUsage:       "give one name of letters, digits and underscores",
			OutUnavailable: "stop and ask the user. This host has no secret verbs yet",
			OutNoSession:   "stop and ask the user for a token with the write:repository scope",
		},
	},
	{
		Noun: "stopwatch", Name: "list",
		Usage:   "vcs stopwatch list [--check]",
		Summary: "List the running stopwatches of the caller.",
		Output:  "One row for each stopwatch: repository, issue, seconds, created.",
		Flags:   []Flag{{"--check", "Exit 2 when a stopwatch runs. It means found, not failed."}},
		Exits:   []string{"0 listed, or none runs", "1 the call failed", "2 with --check, a stopwatch runs", "64 usage error", "69 no usable driver"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "check the token and the host",
			OutUsage:       "this verb takes no number",
			OutUnavailable: "stop and ask the user. This host has no time tracking yet",
		},
	},
	{
		Noun: "git", Name: "commit", Sends: true,
		Usage:   "vcs git [-C <dir>] commit <git commit arguments>...",
		Summary: "Commit with the guard checks. It refuses a hook bypass, a commit on main, a bad branch name, and an AI credit line. It runs the prose lint, the project's scripts/lint-branch-name.sh, and the project checks. Then it runs git commit. A leading -C <dir> works as it does for git.",
		Output:  "The whole output of git commit, after one line for each check.",
		Exits:   []string{"0 committed", "1 a check or git failed", "64 usage error, or a refused flag", "65 the message has a credit line or fails the prose lint", "69 this build can't run git"},
		Hints: map[string]string{
			OutOK:          "run vcs git push to publish the branch",
			OutFail:        "read the check that failed, fix it, then run the command again",
			OutUsage:       "remove the flag that skips the hooks. Fix what the hook reports",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This build can't run git",
		},
	},
	{
		Noun: "git", Name: "push",
		Usage:   "vcs git [-C <dir>] push [<git push arguments>...]",
		Summary: "Push with the guard checks: never a force-push, never a hook bypass. With no arguments it pushes the current branch, and sets the upstream when the branch has none that carries its name.",
		Output:  "The whole output of git push, after one line for each check.",
		Exits:   []string{"0 pushed", "1 a check or git failed", "64 usage error, or a refused flag", "69 this build can't run git"},
		Hints: map[string]string{
			OutOK:          "open or update the pull request",
			OutFail:        "read the check or the git error that vcs printed. To fix a conflict, run vcs git merge origin/main, then push",
			OutUsage:       "never force-push. Add new commits on top instead",
			OutUnavailable: "stop and ask the user. This build can't run git",
		},
	},
	{
		Noun: "git", Name: "merge", Sends: true,
		Usage:   "vcs git [-C <dir>] merge <git merge arguments>...",
		Summary: "Merge with the guard checks: never on main, no AI credit line in a message, no hook bypass. The flags --abort and --quit skip the checks. --continue runs only the wiring check. When the wiring check fails on a clean tree, vcs runs the merge anyway, checks again, and puts the branch back if the check still fails.",
		Output:  "The whole output of git merge, after one line for each check.",
		Exits:   []string{"0 merged", "1 a check, a conflict, or git failed", "64 usage error, or a refused flag", "65 the message fails", "69 this build can't run git"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "on a conflict, fix the files, run git add, then run vcs git merge --continue",
			OutUsage:       "give the git merge arguments, such as origin/main",
			OutLint:        "fix each finding that vcs printed, then run the command again",
			OutUnavailable: "stop and ask the user. This build can't run git",
		},
	},
	{
		Noun: "git", Name: "rebase",
		Usage:   "vcs git [-C <dir>] rebase <git rebase arguments>...",
		Summary: "Rebase a branch that origin doesn't hold yet. It refuses a branch that origin holds, because that needs a force-push. It refuses an interactive rebase.",
		Output:  "The whole output of git rebase, after one line for each check.",
		Exits:   []string{"0 rebased", "1 a check or git failed", "64 usage error, or a refused flag", "69 this build can't run git"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "if origin holds the branch, run vcs git merge origin/main instead",
			OutUsage:       "give the git rebase arguments, without -i",
			OutUnavailable: "stop and ask the user. This build can't run git",
		},
	},
	{
		Noun: "git", Name: "branch",
		Usage:   "vcs git [-C <dir>] branch create|switch|rename|delete|list <args>...",
		Summary: "Make, switch, rename, delete, or list branches. create checks the name with the type/short-description rule, and starts from origin/main unless you name a start point. rename --remote also renames the branch on the server, when the driver can. An open pull request keeps its number. delete refuses the default branch and the current one.",
		Output:  "The whole output of the git command.",
		Exits:   []string{"0 done", "1 a check or git failed", "64 usage error", "69 this build can't run git, or the driver has no branch rename"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "read the check or the git error that vcs printed",
			OutUsage:       "use create <name> [<start>], switch <name>, rename [<old>] <new> [--remote], delete <name> [--remote] [--force], or list",
			OutUnavailable: "stop and ask the user. This build can't run git",
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
		Usage:   "vcs session login [--user <name>]",
		Summary: "Sign in to the Forgejo web interface and keep the session. Some verbs, such as ci rerun, need it because the token API can't do them. In a terminal it asks for the user name and the password, and hides the password. A script sets VCS_PASSWORD and passes --user. The password never goes in an argument.",
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
		Summary: "Keep the " + session.CookieName + " cookie of a browser session, read from standard input. Use it for an account with a second factor.",
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
		Noun: "help", Name: "",
		Usage:   "vcs help [<noun> [<verb>]]",
		Summary: "Show the nouns, the verbs of a noun, or the page of one verb. In a terminal, a bare `vcs help` opens a menu. In a pipe, it prints plain text and never waits for input.",
		Output:  "Plain text: the nouns with one line each, the verbs of a noun, or the usage, flags, exit codes, and one example of a verb. A verb that needs a web session says so.",
		Exits:   []string{"0 printed", "64 no such noun or verb"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "",
			OutUsage:       "run vcs help to list the nouns",
			OutUnavailable: "",
		},
	},
	{
		Noun: "version", Name: "",
		Usage:   "vcs version",
		Summary: "Print the release tag of this binary. `vcs --version` does the same.",
		Output:  "One line: the tag, such as v0.0.6. A dev build prints dev-<sha>.",
		Exits:   []string{"0 printed the tag", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "",
			OutUsage:       "run vcs version with no other argument",
			OutUnavailable: "",
		},
	},
	{
		Noun: "lexi", Name: "",
		Usage:   "vcs lexi [<flags>] [<globs>...]",
		Summary: "Run lexi, the inclusive-language checker, with the flags and the output of the lexi binary. Every argument goes to lexi unchanged, so the flag parser of vcs never sees it. `vcs alex` does the same.",
		Output:  "What lexi prints: the report on stderr, in the format of alex. With --reporter=json, the JSON report.",
		Flags: []Flag{
			{"--stdin", "Read the text from stdin."},
			{"-t, --text", "Treat the input as plain text."},
			{"-l, --html", "Treat the input as HTML."},
			{"--mdx", "Treat the input as Markdown with JSX."},
			{"-q, --quiet", "Print only warnings and errors."},
			{"-w, --why", "Print the sources of a finding."},
			{"-d, --diff", "Ignore unchanged lines."},
			{"--reporter=<name>", "Use the JSON reporter. lexi supports no other."},
		},
		Exits: []string{"The exit code of lexi: 0 no finding, 1 a finding or an error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "",
			OutUsage:       "run vcs lexi --help",
			OutUnavailable: "",
		},
	},
	{
		Noun: "alex", Name: "",
		Usage:   "vcs alex [<flags>] [<globs>...]",
		Summary: "The same command as `vcs lexi`, under the name of alex. It runs lexi with the flags and the output of alex.",
		Output:  "The same as `vcs lexi`.",
		Exits:   []string{"The exit code of lexi: 0 no finding, 1 a finding or an error"},
		Hints: map[string]string{
			OutOK:          "",
			OutFail:        "",
			OutUsage:       "run vcs alex --help",
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
	{
		Noun: "site", Name: "gen",
		Usage:   "vcs site gen <source> <dir>",
		Summary: "Build the documentation site. It reads the hand pages of <source> and the command tree, and writes plain HTML to <dir>.",
		Output:  "One line for each file written.",
		Exits:   []string{"0 built the site", "1 a read or a write failed", "64 usage error"},
		Hints: map[string]string{
			OutOK:          "commit the changed files of the site with the code change",
			OutFail:        "check that the source has guide and reference folders, and that the directory is writable",
			OutUsage:       "give a source and a directory, such as docs/Site site",
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
