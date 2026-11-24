# The `vcs comment list` command

List the comments of an issue or a pull request.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs comment list <n>
```

## Output

One row for each comment, tab-separated: id, user, created, body, updated. The body is on one line, and each newline is the two characters \n.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs comment list 14
501	maria	2026-11-20T10:00:00Z	I can reproduce this.\nSteps below.	2026-11-20T10:05:00Z
```

## Exit codes

- 0 listed
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no comment support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the number |
| ok | use the id with vcs comment edit or vcs comment delete |
| unavailable | stop and ask the user. This host has no comment support yet |
| usage | give the number of an issue or a pull request |
