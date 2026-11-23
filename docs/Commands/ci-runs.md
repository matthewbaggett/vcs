# The `vcs ci runs` command

List CI runs, newest first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs ci runs [--branch <name|#n>] [--sha <sha>] [--status <status>] [--limit <n>]
```

## Output

One row for each run, tab-separated: run-id, index, status, branch, head sha, title, url. A pull request run has `#<n>` in the branch column.

## Flags

| Flag | Meaning |
|---|---|
| `--branch` | Keep the runs of one branch, or of one pull request as `#<n>`. |
| `--sha` | Keep the runs of one commit. A full 40-character sha also filters on the server. |
| `--status` | Keep one status: failure or success. |
| `--limit` | Keep the newest <n> rows. The default is 20. 0 reads every run, up to 1000. |

## Exit codes

- 0 printed the rows (an empty list isn't an error)
- 1 the call failed
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | take a run-id from the first column and run vcs ci jobs <run-id> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | see vcs ci runs for the flags |
