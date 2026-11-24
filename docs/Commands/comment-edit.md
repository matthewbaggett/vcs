# The `vcs comment edit` command

Replace the text of one comment. Find the id with vcs comment list. The prose lint runs first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs comment edit <id> (--body <text> | --body-file <path>)
```

## Output

One line: `ok`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs comment edit 501 --body "Fixed in the pull request."
ok
```

## Flags

| Flag | Meaning |
|---|---|
| `--body` | The new comment text. |
| `--body-file` | A file that holds the new comment text. Use - for standard input. |
| `--trace` | Name the skill and step that sent the text, as skill or skill:step. $VCS_TRACE is the fallback. The command adds a hidden trace line to the end of the body. |

## Exit codes

- 0 edited
- 1 the call failed or the comment isn't found
- 64 usage error, or both or neither of the body flags
- 65 the text fails the prose lint, and the command sends nothing
- 69 no usable driver, or the driver has no comment support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the id again with vcs comment list |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs comment list to check the text |
| unavailable | stop and ask the user. This host has no comment support yet |
| usage | give the comment id and one of --body or --body-file |
