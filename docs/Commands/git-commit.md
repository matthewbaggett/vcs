# The `vcs git commit` command

Commit with the guard checks. It refuses a hook bypass, a commit on main, a bad branch name, and an AI credit line. It runs the prose lint, the project's scripts/lint-branch-name.sh, and the project checks. Then it runs git commit. A leading -C <dir> works as it does for git.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs git [-C <dir>] commit <git commit arguments>...
```

## Output

The whole output of git commit, after one line for each check.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs git commit -m "Add the retry"
vcs: check not-default-branch: ok
vcs: check attribution: ok
[fix/retry 29a8b0f] Add the retry
 2 files changed, 18 insertions(+)
```

## Exit codes

- 0 committed
- 1 a check or git failed
- 64 usage error, or a refused flag
- 65 the message has a credit line or fails the prose lint
- 69 this build can't run git

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the check that failed, fix it, then run the command again |
| lint | fix each finding that vcs printed, then run the command again |
| ok | run vcs git push to publish the branch |
| unavailable | stop and ask the user. This build can't run git |
| usage | remove the flag that skips the hooks. Fix what the hook reports |
