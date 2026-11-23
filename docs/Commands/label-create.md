# The `vcs label create` command

Make one label. The color is six hex digits, with or without a leading #. The command lints the description, and it leaves the name alone because a name is a tag.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs label create <name> <color> [--desc <text>]
```

## Output

One row: `created` and the name.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs label create ci 5319e7 --desc "Build and test"
created	ci
```

## Flags

| Flag | Meaning |
|---|---|
| `--desc` | A one-line description of the label. |

## Exit codes

- 0 created
- 1 the call failed, or the label exists
- 64 usage error, a bad color, or a name or description of more than one line
- 65 a guard rule or the prose lint refuses the description, and the command sends nothing
- 69 no usable driver, or the driver has no label support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run vcs label list to see if the label exists |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs label list to check the label |
| unavailable | stop and ask the user. This host has no label support yet |
| usage | give a one-line name and a color of six hex digits |
