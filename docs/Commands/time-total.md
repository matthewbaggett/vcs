# The `vcs time total` command

Add up the time entries of one issue, or of the whole repository with no number.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs time total [<n>] [--mine]
```

## Output

One row: seconds, then the total as 1h25m0s.

## Flags

| Flag | Meaning |
|---|---|
| `--mine` | Only the entries of the caller. |

## Exit codes

- 0 printed
- 1 the call failed
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number and the token |
| ok | none |
| unavailable | stop and ask the user. This host has no time tracking yet |
| usage | give no number, or one issue number |
