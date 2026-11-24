# The `vcs pr watch` command

Poll the checks and the merge state of one pull request until both settle.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr watch <n> [--interval <seconds>] [--timeout <seconds>] [--remote <name>]
       vcs pr watch <n> --status
```

## Output

One line each time the state changes: time, number, head, ci, and merge. Then a RESULT line. A failed check or a conflict adds a list before the RESULT line. With --status: watching #<n> pid=<pid>, or not watching #<n>.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr watch 12 --interval 30
2026-10-17T10:00:00Z #12 head=29a8b0f ci=pending (1/2 done, 0 failed) merge=clean
2026-10-17T10:01:00Z #12 head=29a8b0f ci=green (2/2 done, 0 failed) merge=clean
RESULT: every check passed and the pull request merges cleanly
```

## Flags

| Flag | Meaning |
|---|---|
| `--interval` | Seconds between two polls. The default is 30. |
| `--timeout` | Seconds to wait before the command gives up. The default is 3600. |
| `--remote` | The git remote that holds the pull request refs. The default is origin. |
| `--status` | Tell whether a watcher of this pull request is alive. Exit 0 if one is, 1 if none is. |

## Exit codes

- 0 every check passed, every required check has a status, and the merge is clean
- 1 a check failed, or a read failed
- 2 the pull request conflicts with its base branch
- 3 the pull request closed or merged
- 4 the timeout came first
- 5 another watcher of this pull request is alive
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the last lines. If a check failed, run vcs ci runs --sha <head sha> --limit 3 and vcs ci log. If conflicted files show, merge main in. Then run vcs pr watch <n> again |
| ok | the pull request is good. Hand it off, or mark it ready with vcs pr ready <n> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | run vcs pr watch <n> with the number of one pull request |
