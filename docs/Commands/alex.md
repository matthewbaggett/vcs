# The `vcs alex` command

An alias of `vcs lexi`. It runs lexi with the flags and the output of alex.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs alex [<flags>] [<globs>...]
```

## Output

The same as `vcs lexi`.

## Example

The output below is an example with made-up data, not a real run.

```text
$ echo "His network looks good" | vcs alex --stdin
<stdin>
  1:1-1:4  warning  `His` may be insensitive, when referring to a person, use `Their`, `Theirs`, `Them` instead  her-him  retext-equality

⚠ 1 warning
```

## Exit codes

- The exit code of lexi: 0 no finding, 1 a finding or an error

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | none |
| ok | none |
| unavailable | none |
| usage | run vcs alex --help |
