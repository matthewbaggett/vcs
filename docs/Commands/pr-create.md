# The `vcs pr create` command

Open a pull request. The command checks the branch name and the AI credit line, and the prose lint runs on the title and body.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr create <title> --head <branch> [--base <branch>] (--body <text> | --body-file <path>) [--draft]
```

## Output

One row: the number and the web address.

## Flags

| Flag | Meaning |
|---|---|
| `--head` | The branch that holds the work. It must match type/short-description. |
| `--base` | The branch to merge into. The default is main. |
| `--body` | The body text. |
| `--body-file` | A file that holds the body. Use - for standard input. |
| `--draft` | Open the pull request as a draft. |

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
| fail | check the token, and that you pushed the branch |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs pr watch <n> until CI ends |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a title, --head, and one of --body or --body-file |
