# The `vcs runners list` command

List the CI runners of the whole instance. The call needs a token of an admin user. Match the runner uuid of a failed job log to a name here.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs runners list
```

## Output

One row for each runner, sorted by id: id, name, status, labels, uuid, version. The labels are a comma list.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs runners list
3	build-1	idle	ubuntu-latest,docker	5f0c9a2e-1111-2222-3333-444455556666	v6.0.1
```

## Exit codes

- 0 listed, or no runner exists
- 1 the call failed
- 64 usage error
- 69 no usable driver
- 77 the server refused the token: it's missing, too weak, or not an admin

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the host and the token |
| nosession | stop and ask an admin for the list, or for a token with the read:admin scope |
| ok | none |
| unavailable | stop and ask the user. This host has no runner list yet |
| usage | this verb takes no argument |
