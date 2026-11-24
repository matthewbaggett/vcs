# The `vcs pr statuses` command

Print the status rows of one commit, one line for each row.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr statuses <sha> [--latest] [--description]
```

## Output

Rows, tab-separated, no header: context, state, address. The server lists every row it ever got, newest first. A state is success, pending, failure, error, or warning. With --description, a fourth column holds the description of the check. An empty list is not an error.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr statuses 5888000a1bc3 --latest
ci / lint	success	https://forge.example/acme/app/actions/runs/9/jobs/1
ci / test	pending	https://forge.example/acme/app/actions/runs/9/jobs/0
```

## Flags

| Flag | Meaning |
|---|---|
| `--latest` | Keep the newest row of each context, sorted by context. |
| `--description` | Add the description as the last column. |

## Exit codes

- 0 printed the rows
- 1 the call failed or the commit isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the commit with git log, then run the command again |
| ok | if a state is failure, run vcs ci runs --sha <sha> --limit 3, then vcs ci jobs <run-id> |
| unavailable | stop and ask the user. This host has no driver for statuses yet |
| usage | run vcs pr statuses <sha> with one commit |
