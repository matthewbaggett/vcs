# The `vcs milestone create` command

Make one milestone. The command refuses the AI credit line, and the prose lint runs on the title and the description first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs milestone create <title> [--due <date>] [--desc <text>]
```

## Output

One row: `created` and the title.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs milestone create v1.4 --due 2027-01-31
created	v1.4
```

## Flags

| Flag | Meaning |
|---|---|
| `--due` | The due date, like 2027-01-31. |
| `--desc` | A description of the milestone. |

## Exit codes

- 0 created
- 1 the call failed, or a milestone with that title exists
- 64 usage error, or a bad date
- 65 a guard rule or the prose lint refuses the text, and the command sends nothing
- 69 no usable driver, or the driver has no milestone support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run vcs milestone list --state all to see if the title exists |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs milestone view <title> to check it |
| unavailable | stop and ask the user. This host has no milestone support yet |
| usage | give a one-line title, and a date like 2027-01-31 if you use --due |
