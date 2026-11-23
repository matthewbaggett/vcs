# The `vcs pr list` command

List pull requests, newest first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr list [--state open|closed|all] [--label <name,name>] [--milestone <title>] [--ci] [--limit <n>]
```

## Output

One row for each pull request, tab-separated: number, title, head branch, base branch, mergeable, ci, labels, milestone. The ci column is empty unless you pass --ci. Labels are comma-separated.

## Flags

| Flag | Meaning |
|---|---|
| `--state` | Keep open, closed, or all pull requests. The default is open. |
| `--label` | Keep the pull requests that carry every label. Separate names with commas. |
| `--milestone` | Keep the pull requests of one milestone. |
| `--ci` | Fill the ci column. It costs one more request for each pull request. |
| `--limit` | Keep the newest <n> rows after the filter. The default keeps all. |

## Exit codes

- 0 printed the rows (an empty list isn't an error)
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no pr list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | take a number from the first column and run vcs pr status <n> |
| unavailable | stop and ask the user. This host has no pr list yet |
| usage | see vcs pr list for the flags |
