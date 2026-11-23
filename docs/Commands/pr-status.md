# The `vcs pr status` command

Print the state of one pull request on one line.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr status <n> [--pushed]
```

## Output

One row, tab-separated: state, draft, mergeable, conflicted files, ci, head sha, url. With --pushed, an eighth column holds the commit time of the head commit, as Zulu time.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr status 12
open	false	true	0	success	5888000a1bc3	https://forge.example/acme/app/pulls/12
```

## Flags

| Flag | Meaning |
|---|---|
| `--pushed` | Add the commit time of the head commit as the last column. It costs one more request. |

## Exit codes

- 0 printed the row
- 1 the call failed or the pull request isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number with vcs pr list, then run the command again |
| ok | if ci is failure, run vcs ci runs --sha <head sha> --limit 3, then vcs ci jobs <run-id> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | run vcs pr status <n> with the number of one pull request |
