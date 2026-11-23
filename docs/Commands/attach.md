# The `vcs attach` command

Upload files to an issue or a pull request, and optionally post one comment that shows them.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs attach <n> <file>... [--comment] [--title <text>]
```

## Output

One row for each file: name, download address. With --comment, a last line `comment posted`.

## Flags

| Flag | Meaning |
|---|---|
| `--comment` | Post one comment that embeds each file as an image. |
| `--title` | The first line of the comment. The default is Screenshots. |

## Exit codes

- 0 uploaded
- 1 an upload or the comment failed
- 64 usage error, or a file isn't a plain file
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token. Run the command again for the files that didn't upload |
| ok | none |
| unavailable | stop and ask the user. This host has no upload route |
| usage | give the number, then one or more plain files |
