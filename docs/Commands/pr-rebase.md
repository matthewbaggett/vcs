# The `vcs pr rebase` command

Tick the rebase box in the body of a Renovate pull request. Renovate then rebases the branch. Forgejo only.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr rebase <n>
```

## Output

One line: `ok` when the command ticks the box, or `already` when the box has a tick.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr rebase 21
ok
```

## Exit codes

- 0 ticked, or already ticked
- 1 the call failed, or the pull request has no rebase box
- 64 usage error
- 69 no usable driver, or the host has no rebase box

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | only a Renovate pull request has a rebase box. Check the number |
| ok | none |
| unavailable | stop and ask the user. This host has no rebase box |
| usage | give the pull request number |
