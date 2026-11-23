# The `vcs label rename` command

Change the name of one label. Find the id with vcs label list.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs label rename <id> <name>
```

## Output

One row: `renamed`, the id, and the new name.

## Exit codes

- 0 renamed
- 1 the call failed or the label isn't found
- 64 usage error
- 69 no usable driver, or the driver has no label support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the id again with vcs label list |
| ok | run vcs label list to check the name |
| unavailable | stop and ask the user. This host has no label support yet |
| usage | give a numeric id and a one-line name |
