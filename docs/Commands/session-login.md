# The `vcs session login` command

Sign in to the Forgejo web interface and keep the session. Some verbs, such as ci rerun, need it because the token API can't do them. In a terminal it asks for the user name and the password, and hides the password. A script sets VCS_PASSWORD and passes --user. The password never goes in an argument.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs session login [--user <name>]
```

## Output

`ok`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ VCS_PASSWORD=... vcs session login --user maria
ok
```

## Flags

| Flag | Meaning |
|---|---|
| `--user` | The user name. VCS_USER is the fallback. |

## Exit codes

- 0 signed in
- 1 the server refused the login, or the account needs a second factor
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | stop and ask the user. For a second factor the user runs vcs session import |
| ok | none |
| unavailable | none |
| usage | stop and ask the user to run this command. Never put a password in a command |
