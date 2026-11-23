# vcs ci rerun

Restart a CI run. The Forgejo API has no rerun route, so this uses the web route with a session.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs ci rerun <run-id|run-address> [--failed]
```

## Output

One row: `rerun` and the run-id.

## Flags

| Flag | Meaning |
|---|---|
| `--failed` | Restart the failed jobs only. |

## Exit codes

- 0 restarted
- 1 the call failed, the run is not found, or --failed found no failed job
- 64 usage error
- 69 no usable driver
- 77 no web session, or the server no longer accepts it

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the run-id again with vcs ci runs |
| nosession | stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat |
| ok | run vcs pr status <n> in a few minutes, or vcs ci runs --limit 1 |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a run-id or the web address of a run |
