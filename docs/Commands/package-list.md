# The `vcs package list` command

List the generic packages of the owner of the repository, and report the old versions. The verb never deletes. No other registry type is reachable.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs package list [<name>] [--stale-days <n>]
```

## Output

One row for each version, newest first within a name: name, version, created, age in days, status. The status is newest, old (at least --stale-days old, default 30), or recent. A name gets a note on stderr when old versions exist.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs package list vendor-tool
vendor-tool	2.0	2026-11-01T10:00:00Z	23	newest
vendor-tool	1.0	2026-09-01T10:00:00Z	84	old
```

## Flags

| Flag | Meaning |
|---|---|
| `--stale-days <n>` | The age in days from which an older version counts as old. The default is 30. |

## Exit codes

- 0 listed, or no package exists
- 1 the call failed
- 64 usage error
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the host, the owner, and the token |
| nosession | stop and ask the user for a token with the read:package scope |
| ok | none |
| unavailable | stop and ask the user. This host has no package verbs yet |
| usage | give at most one package name of lower-case letters, digits, dots, underscores, and dashes |
