# The `vcs pr ready` command

Take a pull request out of draft. A pull request that isn't a draft stays as it is.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr ready <n>
```

## Output

One line: `ready` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr ready 13
ready 13
```

## Exit codes

- 0 ready
- 1 the call failed or the pull request isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number and the token |
| ok | run vcs pr status <n> to check the merge state |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give the pull request number |
