# The `vcs stopwatch list` command

List the running stopwatches of the caller.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs stopwatch list [--check]
```

## Output

One row for each stopwatch: repository, issue, seconds, created.

## Flags

| Flag | Meaning |
|---|---|
| `--check` | Exit 2 when a stopwatch runs. It means found, not failed. |

## Exit codes

- 0 listed, or none runs
- 1 the call failed
- 2 with --check, a stopwatch runs
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the host |
| ok | none |
| unavailable | stop and ask the user. This host has no time tracking yet |
| usage | this verb takes no number |
