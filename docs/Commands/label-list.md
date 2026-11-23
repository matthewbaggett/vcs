# The `vcs label list` command

List the labels of the repository.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs label list
```

## Output

One row for each label, tab-separated: id, name, color, description.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs label list
1	bug	d73a4a	Something is broken
2	docs	0075ca	Documentation
```

## Exit codes

- 0 listed
- 1 the call failed
- 64 usage error
- 69 no usable driver, or the driver has no label support

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the host |
| ok | use the id with vcs label rename |
| unavailable | stop and ask the user. This host has no label support yet |
| usage | give no argument |
