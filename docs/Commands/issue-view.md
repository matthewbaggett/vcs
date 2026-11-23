# The `vcs issue view` command

Print one issue with its body and its comments.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue view <n>
```

## Output

A first row, tab-separated: number, title, state, labels, milestone. Then a blank line and the body. Then each comment: a blank line, a row of `comment`, the writer, and the date, and the text.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue view 14
14	Fix the retry	open	bug	v1.2

The client gives up after one failed call.

comment	maria	2026-11-20
I can reproduce this.
```

## Exit codes

- 0 printed the issue
- 1 the call failed or the issue isn't found
- 64 usage error
- 69 no usable driver, or the driver has no issue view

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number with vcs issue list |
| ok | none |
| unavailable | stop and ask the user. This host has no issue view yet |
| usage | run vcs issue view <n> with the number of one issue |
