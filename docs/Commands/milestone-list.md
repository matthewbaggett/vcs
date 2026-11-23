# The `vcs milestone list` command

List the milestones of the repository.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs milestone list [--state open|closed|all]
```

## Output

One row for each milestone, tab-separated: id, title, state, open issues, closed issues, due date. The due date looks like 2026-12-01, or the field is empty.

## Flags

| Flag | Meaning |
|---|---|
| `--state` | open, closed, or all. The default is open. |

## Exit codes

- 0 listed
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no milestone support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the host |
| ok | use the title with vcs issue edit --milestone |
| unavailable | stop and ask the user. This host has no milestone support yet |
| usage | use --state open, closed, or all |
