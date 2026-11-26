# The `vcs pr draft` command

Turn a pull request back into a draft. A draft stays a draft.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr draft <n>
```

## Output

One line: `draft` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr draft 13
draft 13
```

## Exit codes

- 0 draft
- 1 the call failed, the pull request isn't found, or the pull request has the state closed or merged
- 64 usage error
- 69 no usable driver, or the driver has no draft support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number and the token |
| ok | run vcs pr status <n> to check the draft flag |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give the pull request number |
