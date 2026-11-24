# The `vcs issue top` command

Print the highest index of any issue or pull request.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue top
```

## Output

One number: the highest index in any state, or 0 when the repository has none.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue top
183
```

## Exit codes

- 0 printed the number
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no index list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | keep the number as the mark, then run vcs issue newer <mark> in the next round |
| unavailable | stop and ask the user. This host has no index list yet |
| usage | run vcs issue top with no argument |
