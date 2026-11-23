# The `vcs board add` command

Put issues or pull requests on a board. Forgejo needs a web session for this.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs board add <board-id> <n>...
```

## Output

One line: `ok`.

## Exit codes

- 0 added
- 1 the call failed or an issue isn't found
- 64 usage error
- 69 no usable driver
- 77 no web session, or the server no longer accepts it

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the board id and the numbers |
| nosession | stop and ask the user to run vcs session login, or vcs session import. Never ask for the password in chat |
| ok | none |
| unavailable | stop and ask the user. This host has no board support |
| usage | give the board id, then one or more numbers |
