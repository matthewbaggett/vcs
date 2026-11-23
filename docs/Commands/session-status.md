# The `vcs session status` command

Tell whether the kept session is still good.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs session status
```

## Output

`ok`.

## Exit codes

- 0 the session is good
- 64 usage error
- 77 no session, or the server no longer accepts it

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | none |
| nosession | stop and ask the user to run vcs session login, or vcs session import |
| ok | none |
| unavailable | none |
| usage | run vcs session status with no argument |
