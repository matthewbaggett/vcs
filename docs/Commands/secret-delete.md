# The `vcs secret delete` command

Delete one Actions secret.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs secret delete <name>
```

## Output

ok.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs secret delete DEPLOY_KEY
ok
```

## Exit codes

- 0 deleted
- 1 the call failed, or the secret does not exist
- 64 usage error
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the name with vcs secret list |
| nosession | stop and ask the user for a token with the write:repository scope |
| ok | none |
| unavailable | stop and ask the user. This host has no secret verbs yet |
| usage | give one name of letters, digits and underscores |
