# The `vcs version` command

Print the release tag of this binary. `vcs --version` does the same.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs version
```

## Output

One line: the tag, such as v0.0.6. A dev build prints dev-<sha>.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs version
v0.0.6
```

## Exit codes

- 0 printed the tag
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | none |
| ok | none |
| unavailable | none |
| usage | run vcs version with no other argument |
