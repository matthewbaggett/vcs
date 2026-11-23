# The `vcs pr tick` command

Read every open pull request once. Merge each one that isn't a draft, merges clean, targets main, and has green checks. Report each one that needs a person. Run it only when the owner has said to merge ready pull requests.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr tick [--dry-run]
```

## Output

One row for each pull request that needs attention: a word, the number, the head sha. The words: `MERGED`, `MERGED-BUT-NOT-LANDED`, `MERGE-FAILED`, `RED`, `NOT-MERGEABLE` (with ci=<state>), `DRAFT-GREEN`, `DRAFT-RED`, and `WOULD-MERGE`. With nothing to report, it prints `QUIET`.

## Flags

| Flag | Meaning |
|---|---|
| `--dry-run` | Merge nothing. Print `WOULD-MERGE` for each pull request that's ready. |

## Exit codes

- 0 the pass ran. The rows hold the result
- 1 the call to list the pull requests failed
- 64 usage error
- 69 no usable driver, or the driver has no pr list or pr merge

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | act on each row: fix a `RED` or `NOT-MERGEABLE` pull request, mark a `DRAFT-GREEN` one ready, and check a `MERGE-FAILED` one with vcs pr status <n> |
| unavailable | stop and ask the user. This host has no pr list or pr merge yet |
| usage | run vcs pr tick with no argument, or with --dry-run |
