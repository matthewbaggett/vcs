# The `vcs pr heal` command

Mark and unmark the pull requests that a worker heals now, with the label `➕healing`. The environment variable VCS_HEAL_LABEL changes the name. The sweep clears the mark of each pull request that closed, has no live worker, or is healthy.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr heal ensure | add <n> | remove <n> | sweep --live <n,n,...|none>
```

## Output

`ensure` prints `created` or `exists`. `add` prints `ok`. `remove` prints `ok`, or `absent` when the pull request has no mark. `sweep` prints one row for each mark it cleared: `cleared`, the number, and the reason (`closed`, `no-worker`, or `healthy`).

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr heal sweep --live 12,13
cleared	9	closed
cleared	10	no-worker
```

## Flags

| Flag | Meaning |
|---|---|
| `--live` | For sweep: the numbers of the pull requests that a worker holds now, separated by commas, or `none`. |

## Exit codes

- 0 done
- 1 a call failed
- 64 usage error
- 69 no usable driver, or the driver has no label, issue edit, or pr list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | run the command again. If it fails twice, stop and report the error |
| ok | the mark is in the state you asked for |
| unavailable | stop and ask the user. This host has no label, issue edit, or pr list yet |
| usage | use `ensure`, `add <n>`, `remove <n>`, or `sweep --live <numbers>`. Separate the numbers with commas, or give `none` |
