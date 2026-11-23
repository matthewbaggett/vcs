# vcs pr ready

Take a pull request out of draft. A pull request that is not a draft stays as it is.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs pr ready <n>
```

## Output

One line: `ready` and the number.

## Exit codes

- 0 ready
- 1 the call failed or the pull request is not found
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
