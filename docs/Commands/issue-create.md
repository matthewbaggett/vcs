# The `vcs issue create` command

Open an issue. The command refuses the AI credit line, and the prose lint runs on the title and body.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue create <title> (--body <text> | --body-file <path>)
```

## Output

One row: the number and the web address.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue create "Fix the retry" --body-file body.md
14	https://forge.example/acme/app/issues/14
```

## Flags

| Flag | Meaning |
|---|---|
| `--body` | The body text. |
| `--body-file` | A file that holds the body. Use - for standard input. |

## Exit codes

- 0 opened
- 1 the call failed
- 64 usage error
- 65 a guard rule or the prose lint refuses the text, and the command sends nothing
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token, then run the command again |
| lint | fix each finding that vcs printed, then run the command again |
| ok | none |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a title and one of --body or --body-file |
