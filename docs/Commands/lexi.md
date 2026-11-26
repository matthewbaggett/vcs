# The `vcs lexi` command

Run lexi, the inclusive-language checker, with the flags and the output of the lexi binary. Every argument goes to lexi unchanged, so the flag parser of vcs never sees it. `vcs alex` does the same.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs lexi [<flags>] [<globs>...]
```

## Output

What lexi prints: the report on stderr, in the format of alex. With --reporter=json, the JSON report.

## Example

The output below is an example with made-up data, not a real run.

```text
$ echo "His network looks good" | vcs lexi --stdin
<stdin>
  1:1-1:4  warning  `His` may be insensitive, when referring to a person, use `Their`, `Theirs`, `Them` instead  her-him  retext-equality

⚠ 1 warning
```

## Flags

| Flag | Meaning |
|---|---|
| `--stdin` | Read the text from stdin. |
| `-t, --text` | Treat the input as plain text. |
| `-l, --html` | Treat the input as HTML. |
| `--mdx` | Treat the input as Markdown with JSX. |
| `-q, --quiet` | Print only warnings and errors. |
| `-w, --why` | Print the sources of a finding. |
| `-d, --diff` | Ignore unchanged lines. |
| `--reporter=<name>` | Use the JSON reporter. lexi supports no other. |

## Exit codes

- The exit code of lexi: 0 no finding, 1 a finding or an error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | none |
| ok | none |
| unavailable | none |
| usage | run vcs lexi --help |
