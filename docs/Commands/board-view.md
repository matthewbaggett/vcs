# The `vcs board view` command

Show the cards of one board.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs board view <board-id>
```

## Output

One row for each card: column id, column title, item id, number, title. A column with no cards gives no row.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs board view 5
11	To do	201	16	Add the board
12	Doing	202	14	Fix the retry
```

## Exit codes

- 0 shown
- 1 the call failed or the board isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the board id again with vcs board list |
| ok | move a card with vcs board move <board-id> <column> <n> |
| unavailable | stop and ask the user. This host has no board support |
| usage | give one board id |
