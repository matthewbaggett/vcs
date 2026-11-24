# The `vcs package delete` command

Delete one version of a generic package. The verb refuses the newest version unless you pass --allow-newest.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs package delete <name> <version> [--allow-newest]
```

## Output

ok.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs package delete vendor-tool 1.0
ok
```

## Flags

| Flag | Meaning |
|---|---|
| `--allow-newest` | Delete the newest version on purpose. |

## Exit codes

- 0 deleted
- 1 a call failed, the version is missing, or it is the newest
- 64 usage error
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | list the versions with vcs package list <name> |
| nosession | stop and ask the user for a token with the write:package scope |
| ok | none |
| unavailable | stop and ask the user. This host has no package verbs yet |
| usage | give a name and a version |
