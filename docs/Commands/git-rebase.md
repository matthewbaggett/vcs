# The `vcs git rebase` command

Rebase a branch that origin doesn't hold yet. It refuses a branch that origin holds, because that needs a force-push. It refuses an interactive rebase.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs git [-C <dir>] rebase <git rebase arguments>...
```

## Output

The whole output of git rebase, after one line for each check.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs git rebase origin/main
vcs: check not-default-branch: ok
Successfully rebased and updated refs/heads/fix/retry.
```

## Exit codes

- 0 rebased
- 1 a check or git failed
- 64 usage error, or a refused flag
- 69 this build can't run git

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | if origin holds the branch, run vcs git merge origin/main instead |
| ok | none |
| unavailable | stop and ask the user. This build can't run git |
| usage | give the git rebase arguments, without -i |
