# The `vcs pr close` command

Close a pull request with no merge.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr close <n>
```

## Output

One line: `closed` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr close 13
closed 13
```

## Exit codes

- 0 closed
- 1 the call failed or the pull request isn't found
- 64 usage error
- 69 no usable driver, or the driver has no close support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number and the token |
| ok | none |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give the pull request number |
