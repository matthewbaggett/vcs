# The `vcs site gen` command

Build the documentation site. It reads the hand pages of <source> and the command tree, and writes plain HTML to <dir>.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs site gen <source> <dir>
```

## Output

One line for each file written.

## Exit codes

- 0 built the site
- 1 a read or a write failed
- 64 usage error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check that the source has guide and reference folders, and that the directory is writable |
| ok | commit the changed files of the site with the code change |
| unavailable | none |
| usage | give a source and a directory, such as docs/Site site |
