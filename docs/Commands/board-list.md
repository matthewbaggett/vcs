# The `vcs board list` command

List the project boards of the repository.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs board list [--state open|closed]
```

## Output

One row for each board: id, title, address.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs board list --state open
5	Roadmap	https://forge.example/acme/app/projects/5
```

## Flags

| Flag | Meaning |
|---|---|
| `--state` | open or closed. The default is open. |

## Exit codes

- 0 listed
- 1 the call failed
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the token and the host |
| ok | none |
| unavailable | stop and ask the user. This host has no board support |
| usage | use --state open or --state closed |
