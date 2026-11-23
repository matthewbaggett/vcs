# The `vcs board move` command

Move cards to a column, by column id or title. Forgejo needs a web session for this.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs board move <board-id> <column> <n>...
```

## Output

One line: `ok`.

## Exit codes

- 0 moved
- 1 the call failed, or the board, column or issue isn't found
- 64 usage error
- 69 no usable driver
- 77 no web session, or the server no longer accepts it

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the column ids with vcs board view <board-id> |
| nosession | stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat |
| ok | none |
| unavailable | stop and ask the user. This host has no board support |
| usage | give the board id, the column, then one or more numbers |
