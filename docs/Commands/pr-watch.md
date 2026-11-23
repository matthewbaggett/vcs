# The `vcs pr watch` command

Poll the checks and the merge state of one pull request until both are good.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr watch <n> [--interval <seconds>] [--timeout <seconds>]
```

## Output

One row each time the state changes, in the format of vcs pr status.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr watch 12 --interval 30
open	true	true	0	pending	29a8b0f4c2d1	https://forge.example/acme/app/pulls/12
open	false	true	0	success	29a8b0f4c2d1	https://forge.example/acme/app/pulls/12
```

## Flags

| Flag | Meaning |
|---|---|
| `--interval` | Seconds between two polls. The default is 30. |
| `--timeout` | Seconds to wait before the command gives up. The default is 3600. |

## Exit codes

- 0 the pull request merged, or its checks are green and its merge state is clean
- 1 a check failed, the pull request has a conflict or closed, the call failed, or the wait timed out
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the last row. If ci is failure, run vcs ci runs --sha <head sha> --limit 3 and vcs ci log. If conflicted files show, merge main in. Then run vcs pr watch <n> again |
| ok | the pull request is good. Hand it off, or mark it ready with vcs pr ready <n> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | run vcs pr watch <n> with the number of one pull request |
