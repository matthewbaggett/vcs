# The `vcs milestone view` command

Show one milestone. Give the id or the title. A closed milestone counts too.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs milestone view <id|title>
```

## Output

One row: id, title, state, open issues, closed issues, due date, description. The description fits on one line, and each newline shows as the two characters \n.

## Exit codes

- 0 shown
- 1 the call failed or the milestone isn't found
- 64 usage error
- 69 no usable driver, or the driver has no milestone support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run vcs milestone list --state all to see the titles |
| ok | none |
| unavailable | stop and ask the user. This host has no milestone support yet |
| usage | give an id or a one-line title |
