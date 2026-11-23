# The `vcs pr merge` command

Merge a pull request. Run it only when the owner has said to merge that pull request. The command doesn't check the checks, so run vcs pr status first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs pr merge <n> [--method merge|rebase|rebase-merge|squash] [--delete-branch]
```

## Output

One line: `merged` and the number.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs pr merge 13 --method squash
merged 13
```

## Flags

| Flag | Meaning |
|---|---|
| `--method` | How to merge. The default is merge. |
| `--delete-branch` | Delete the head branch after the merge. |

## Exit codes

- 0 merged
- 1 the call failed, the server refused the merge, or the pull request already merged
- 64 usage error
- 69 no usable driver, or the driver has no pr merge

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the message. Check the merge state with vcs pr status <n> |
| lint | the command sends no text, so this exit doesn't occur. Check the arguments |
| ok | run vcs pr status <n> to confirm the state says merged |
| unavailable | stop and ask the user. This host has no pr merge yet |
| usage | give the pull request number. The method is merge, rebase, rebase-merge or squash |
