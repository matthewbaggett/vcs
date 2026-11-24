# The `vcs pr edit` command

Change the title, the body or the base branch of a pull request. The command refuses the AI credit line, and the prose lint runs on the new text. A draft stays a draft.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr edit <n> [--title <text>] [--body <text> | --body-file <path>] [--base <branch>]
```

## Output

One line: `edited` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr edit 13 --title "Add the retry loop"
edited 13
```

## Flags

| Flag | Meaning |
|---|---|
| `--title` | The new title. |
| `--body` | The new body text. |
| `--body-file` | A file that holds the new body. Use - for standard input. |
| `--base` | The new base branch. |
| `--trace` | Name the skill and step that sent the text, as skill or skill:step. $VCS_TRACE is the fallback. The command adds a hidden trace line to the end of the body. |

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
| usage | give the pull request number and at least one of --title, --body, --body-file or --base |
