# The `vcs git branch` command

Make, switch, rename, delete, or list branches. create checks the name with the type/short-description rule, and starts from origin/main unless you name a start point. rename --remote also renames the branch on the server, when the driver can. An open pull request keeps its number. delete refuses the default branch and the current one.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs git branch create|switch|rename|delete|list <args>...
```

## Output

The whole output of the git command.

## Exit codes

- 0 done
- 1 a check or git failed
- 64 usage error
- 69 this build can't run git, or the driver has no branch rename

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | read the check or the git error that vcs printed |
| ok | none |
| unavailable | stop and ask the user. This build can't run git |
| usage | use create <name> [<start>], switch <name>, rename [<old>] <new> [--remote], delete <name> [--remote] [--force], or list |
