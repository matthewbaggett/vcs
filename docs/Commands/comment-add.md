# The `vcs comment add` command

Post one comment on an issue or a pull request. The prose lint runs first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs comment add <n> (--body <text> | --body-file <path>)
```

## Output

One line: `comment posted`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs comment add 14 --body "Fixed in the pull request."
comment posted
```

## Flags

| Flag | Meaning |
|---|---|
| `--body` | The comment text. |
| `--body-file` | A file that holds the comment text. Use - for standard input. |
| `--trace` | Name the skill and step that sent the text, as skill or skill:step. $VCS_TRACE is the fallback. The command adds a hidden trace line to the end of the body. |

## Exit codes

- 0 posted
- 1 the call failed
- 64 usage error, or both or neither of the body flags
- 65 the text fails the prose lint, and the command sends nothing
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the number, then run the command again |
| lint | fix each finding that vcs printed, then run the command again |
| ok | none |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give the number and one of --body or --body-file |
