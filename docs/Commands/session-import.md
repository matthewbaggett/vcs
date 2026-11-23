# The `vcs session import` command

Keep the i_like_gitea cookie of a browser session, read from standard input. Use it for an account with a second factor.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs session import < cookie-value
```

## Output

`ok`.

## Exit codes

- 0 saved
- 1 the server didn't accept the cookie
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | stop and ask the user for a fresh cookie |
| ok | none |
| unavailable | none |
| usage | give the cookie value on standard input |
