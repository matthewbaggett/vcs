# The `vcs time add` command

Log time on an issue or a pull request. The duration is like 1h25m, 90m, or 45s.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs time add <n> <duration>
```

## Output

One row: `ok`, the new total in seconds, and the new total as 1h25m0s. If vcs can't read the total, one line `ok`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs time add 14 1h30m
ok	5400	1h30m0s
```

## Exit codes

- 0 logged
- 1 the call failed or the issue isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run vcs time list <n> to see if the entry exists, before you run the command again |
| ok | vcs logged the time once. Don't run the command again to check it |
| unavailable | stop and ask the user. This host has no time tracking yet |
| usage | give the number, then a duration of hours, minutes and seconds, like 1h30m |
