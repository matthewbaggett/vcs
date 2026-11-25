# The `vcs help` command

Show the nouns, the verbs of a noun, or the page of one verb. In a terminal, a bare `vcs help` opens a menu. In a pipe, it prints plain text and never waits for input.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs help [<noun> [<verb>]]
```

## Output

Plain text: the nouns with one line each, the verbs of a noun, or the usage, flags, exit codes, and one example of a verb. A verb that needs a web session says so.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs help session login
usage: vcs session login [--user <name>]
...
```

## Exit codes

- 0 printed
- 64 no such noun or verb

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | none |
| ok | none |
| unavailable | none |
| usage | run vcs help to list the nouns |
