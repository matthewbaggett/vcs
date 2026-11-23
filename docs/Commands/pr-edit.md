# The `vcs pr edit` command

Change the title or the body of a pull request. The command refuses the AI credit line, and the prose lint runs on the new text. A draft stays a draft.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr edit <n> [--title <text>] [--body <text> | --body-file <path>]
```

## Output

One line: `edited` and the number.

## Flags

| Flag | Meaning |
|---|---|
| `--title` | The new title. |
| `--body` | The new body text. |
| `--body-file` | A file that holds the new body. Use - for standard input. |

## Exit codes

- 0 edited
- 1 the call failed or the pull request isn't found
- 64 usage error
- 65 a guard rule or the prose lint refuses the text, and the command sends nothing
- 69 no usable driver, or the driver has no pr edit

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number and the token |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs pr status <n> to check the pull request |
| unavailable | stop and ask the user. This host has no pr edit yet |
| usage | give the pull request number and at least one of --title, --body or --body-file |
