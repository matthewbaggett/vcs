# The `vcs comment delete` command

Delete one comment. Find the id with vcs comment list. A deleted comment can't come back.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs comment delete <id>
```

## Output

One line: `ok`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs comment delete 501
ok
```

## Exit codes

- 0 deleted
- 1 the call failed or the comment isn't found
- 64 usage error
- 69 no usable driver, or the driver has no comment support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the id again with vcs comment list |
| ok | none |
| unavailable | stop and ask the user. This host has no comment support yet |
| usage | give the numeric comment id |
