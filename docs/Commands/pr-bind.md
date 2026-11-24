# The `vcs pr bind` command

Find the pull request of a branch and print what a session needs to start work on it: the state, the CI word, and each red job. It only reads. It starts no watcher. With two open pull requests on one branch, it takes the newest.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr bind [<number>] [--branch <name>]
```

## Output

A pr row: pr, number, state, draft, mergeable, conflicted files, ci, head sha, url. Then one red row for each red job: red, run id, job id, job name, first error. A line on stderr says how to wait: vcs pr watch.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr bind
pr	12	open	false	true		failure	abc1234def	https://git.example/o/r/pulls/12
red	301	902	test	FAIL TestX
```

## Flags

| Flag | Meaning |
|---|---|
| `--branch <name>` | The head branch to search for. The default is the current branch of the checkout. |

## Exit codes

- 0 printed the rows
- 1 no open pull request has the branch, or a call failed
- 64 usage error
- 69 no usable driver, or the driver has no pr list

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the branch name, and that the pull request is open |
| ok | read each red row with vcs ci log <job id>, or wait with vcs pr watch <number> |
| unavailable | stop and ask the user. This host has no pr list yet |
| usage | give one number, or a --branch, not both |
