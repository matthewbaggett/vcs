# The `vcs package upload` command

Upload one file as a version of a generic package, and link the package to the repository. An upload never replaces: an existing version stops the verb. The file must be a plain file, not a link, not empty, and within the size limit.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs package upload <name> <version> <file> [--sha256 <hex>] [--max-mib <n>] [--stale-days <n>] [--dry-run]
```

## Output

A first row: ok, the download address, the checksum of the file. With --dry-run the first word is dry-run, and the verb sends nothing. Then the list rows of the same name, so the old versions show at once.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs package upload vendor-tool 2.0 vendor-tool-2.0.tgz
ok	https://git.example/api/packages/o/generic/vendor-tool/2.0/vendor-tool-2.0.tgz	9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
```

## Flags

| Flag | Meaning |
|---|---|
| `--sha256 <hex>` | Stop with exit 1 unless the file has this checksum. |
| `--max-mib <n>` | The size limit in MiB. The default is 512. |
| `--stale-days <n>` | The age in days from which an older version counts as old. The default is 30. |
| `--dry-run` | Check everything and print the address and the checksum. Send nothing. |

## Exit codes

- 0 uploaded, or the dry run passed
- 1 a call failed, the version exists, or the checksum differs
- 64 usage error, or a name, a version, or a file that the scope rules refuse
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the message. An existing version needs a new version number, or a delete on purpose |
| nosession | stop and ask the user for a token with the write:package scope |
| ok | none |
| unavailable | stop and ask the user. This host has no package verbs yet |
| usage | give a name, a version, and a plain file |
