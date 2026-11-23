# The `vcs git push` command

Push with the guard checks: never a force-push, never a hook bypass. With no arguments it pushes the current branch, and sets the upstream when the branch has none that carries its name.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs git push [<git push arguments>...]
```

## Output

The whole output of git push, after one line for each check.

## Exit codes

- 0 pushed
- 1 a check or git failed
- 64 usage error, or a refused flag
- 69 this build can't run git

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the check or the git error that vcs printed. To fix a conflict, run vcs git merge origin/main, then push |
| ok | open or update the pull request |
| unavailable | stop and ask the user. This build can't run git |
| usage | never force-push. Add new commits on top instead |
