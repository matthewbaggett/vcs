# The `vcs ci artifact get` command

Download one artifact of a CI run, or every artifact with --all. The server sends each artifact as a zip archive, and the command writes it as it is. It doesn't unpack it.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs ci artifact get <run-id|run-address> <name> [-o <path>]
       vcs ci artifact get <run-id|run-address> --all [-o <dir>]
```

## Output

One row for each file written, tab-separated: path, uncompressed size in bytes.

## Flags

| Flag | Meaning |
|---|---|
| `--all` | Download every artifact that hasn't expired. Give no name. |
| `-o` | The file to write. The default is `<name>.zip` in the current directory, and the command refuses to replace a file that exists. With --all, the path is an existing directory that takes each `<name>.zip`. |

## Exit codes

- 0 wrote the files
- 1 the call failed, the run or the artifact isn't found, the artifact expired, or a default file exists
- 64 usage error
- 69 no usable driver, or the host has no artifact route

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | list the names with vcs ci artifacts <run-id>, then run the command again |
| ok | unpack the zip with unzip into a new empty directory, then read the files |
| unavailable | stop and ask the user. This host has no artifact route yet |
| usage | give a run-id and one name, or a run-id and --all |
