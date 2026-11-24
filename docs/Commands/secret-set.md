# The `vcs secret set` command

Create or replace one Actions secret. The value never comes from an argument. Read it from a file, from standard input with -, or from an environment variable. The verb cuts the trailing newlines and prints no value.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs secret set <name> (--value-file <file|-> | --from-env <VAR>)
```

## Output

ok.

## Example

The output below is an example with made-up data, not a real run.

```text
$ printf '%s' "$TOKEN" | vcs secret set DEPLOY_KEY --value-file -
ok
```

## Flags

| Flag | Meaning |
|---|---|
| `--value-file <file|->` | Read the value from a file, or from standard input for -. |
| `--from-env <VAR>` | Read the value from this environment variable. |

## Exit codes

- 0 set
- 1 the call failed
- 64 usage error, a bad name, or an empty value. Nothing is sent
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the host, the repository, and the token |
| nosession | stop and ask the user for a token with the write:repository scope |
| ok | none |
| unavailable | stop and ask the user. This host has no secret verbs yet |
| usage | give a name of letters, digits and underscores, and exactly one of --value-file or --from-env. The value must not be empty |
