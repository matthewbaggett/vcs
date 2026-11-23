# The `vcs session logout` command

Delete the kept session of this host.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs session logout
```

## Output

`ok`.

## Exit codes

- 0 deleted, or there was none
- 1 the delete failed
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the permissions of the config directory |
| ok | none |
| unavailable | none |
| usage | run vcs session logout with no argument |
