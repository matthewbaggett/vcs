# The `vcs secret list` command

List the Actions secrets of the repository. The server never returns a value.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs secret list
```

## Output

One row for each secret, sorted by name: name, created.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs secret list
DEPLOY_KEY	2026-11-01T10:00:00Z
```

## Exit codes

- 0 listed, or none exists
- 1 the call failed
- 64 usage error
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
| usage | this verb takes no argument |
