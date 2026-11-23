# vcs session login

Sign in to the web side of the forge and keep the session. The password comes from VCS_PASSWORD, never from an argument.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
VCS_PASSWORD=... vcs session login --user <name>
```

## Output

`ok`.

## Flags

| Flag | Meaning |
|---|---|
| `--user` | The user name. VCS_USER is the fallback. |

## Exit codes

- 0 signed in
- 1 the login was refused, or the account needs a second factor
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | stop and ask the user. For a second factor the user runs vcs session import |
| ok | none |
| unavailable | none |
| usage | stop and ask the user to run this command. Never put a password in a command |
