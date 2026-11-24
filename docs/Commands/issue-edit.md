# The `vcs issue edit` command

Change the title, the body, the state, the milestone, or the labels of an issue. The command refuses the AI credit line, and the prose lint runs on the new text. An unknown label or milestone stops the command before anything changes.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue edit <n> [--title <text>] [--body <text> | --body-file <path>] [--state open|closed] [--milestone <title|id|none>] [--add-label <a,b>] [--rm-label <a,b>]
```

## Output

One line: `edited` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue edit 14 --state closed --add-label done
edited 14
```

## Flags

| Flag | Meaning |
|---|---|
| `--title` | The new title. |
| `--body` | The new body text. |
| `--body-file` | A file that holds the new body. Use - for standard input. |
| `--trace` | Name the skill and step that sent the text, as skill or skill:step. $VCS_TRACE is the fallback. The command adds a hidden trace line to the end of the body. |
| `--state` | open to reopen the issue, or closed to close it. |
| `--milestone` | A milestone title or id. Use none to clear it. |
| `--add-label` | Labels to add, separated by commas. |
| `--rm-label` | Labels to remove, separated by commas. |

## Exit codes

- 0 edited
- 1 the call failed, or the issue, a label or the milestone isn't found
- 64 usage error
- 65 a guard rule or the prose lint refuses the text, and the command sends nothing
- 69 no usable driver, or the driver has no issue edit

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number, the label names and the token |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs issue view <n> to check the issue |
| unavailable | stop and ask the user. This host has no issue edit yet |
| usage | give the issue number and at least one change. The state is open or closed |
