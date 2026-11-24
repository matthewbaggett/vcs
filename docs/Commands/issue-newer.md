# The `vcs issue newer` command

List the issues with an index higher than a mark, newest first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue newer <index> [--state open|all] [--kind issue|any]
```

## Output

One row for each item, tab-separated, no header: index, kind, user, title. The kind is issue or pr. The user is the login that opened the item. Issues and pull requests share one index space, so a mark moves on a pull request. An empty list isn't an error.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue newer 180 --kind any
183	pr	matt	Add the board
181	issue	bot	Fix the retry
```

## Flags

| Flag | Meaning |
|---|---|
| `--state` | Keep open items, or all. The default is open. |
| `--kind` | Keep issues only, or any item with a pull request too. The default is issue. |

## Exit codes

- 0 printed the rows
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no index list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | take an index from the first column and run vcs issue view <n>, or keep the highest index as the new mark |
| unavailable | stop and ask the user. This host has no index list yet |
| usage | run vcs issue newer <index> with a whole number, 0 or more |
