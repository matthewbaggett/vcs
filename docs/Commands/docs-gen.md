# vcs docs gen

Write one Markdown page for each verb of the command tree.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs docs gen <dir>
```

## Output

One line for each page written.

## Exit codes

- 0 wrote the pages
- 1 a write failed
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check that the directory exists and is writable |
| ok | commit the changed pages with the code change |
| unavailable | none |
| usage | give one directory, such as docs/Commands |
