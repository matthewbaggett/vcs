# The `vcs git merge` command

Merge with the guard checks: never on main, no AI credit line in a message, no hook bypass. The flags --abort and --quit skip the checks. --continue runs only the wiring check. When the wiring check fails on a clean tree, vcs runs the merge anyway, checks again, and puts the branch back if the check still fails.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs git [-C <dir>] merge <git merge arguments>...
```

## Output

The whole output of git merge, after one line for each check.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs git merge origin/main
vcs: check not-default-branch: ok
Already up to date.
```

## Exit codes

- 0 merged
- 1 a check, a conflict, or git failed
- 64 usage error, or a refused flag
- 65 the message fails
- 69 this build can't run git

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | on a conflict, fix the files, run git add, then run vcs git merge --continue |
| lint | fix each finding that vcs printed, then run the command again |
| ok | none |
| unavailable | stop and ask the user. This build can't run git |
| usage | give the git merge arguments, such as origin/main |
