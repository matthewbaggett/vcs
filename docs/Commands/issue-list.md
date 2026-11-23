# The `vcs issue list` command

List issues, newest first. The list holds no pull request.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue list [--state open|closed|all] [--label <name,name>] [--milestone <title>] [--limit <n>]
```

## Output

One row for each issue, tab-separated: number, title, state, labels, milestone. Labels are comma-separated.

## Flags

| Flag | Meaning |
|---|---|
| `--state` | Keep open, closed, or all issues. The default is open. |
| `--label` | Keep the issues that carry every label. Separate names with commas. |
| `--milestone` | Keep the issues of one milestone. |
| `--limit` | Keep the newest <n> rows after the filter. The default keeps all. |

## Exit codes

- 0 printed the rows (an empty list isn't an error)
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no issue list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | take a number from the first column and run vcs issue view <n> |
| unavailable | stop and ask the user. This host has no issue list yet |
| usage | see vcs issue list for the flags |
