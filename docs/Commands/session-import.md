# vcs session import

Keep the i_like_gitea cookie of a browser session, read from standard input. Use it for an account with a second factor.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs session import < cookie-value
```

## Output

`ok`.

## Exit codes

- 0 saved
- 1 the server did not accept the cookie
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | stop and ask the user for a fresh cookie |
| ok | none |
| unavailable | none |
| usage | give the cookie value on standard input |
