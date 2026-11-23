# Hand-off

Decisions and open items for the `vcs` build. A session reads this first and updates it last.

## Decisions

| Decision | Source |
|---|---|
| One Go binary, `vcs`, replaces `tea`, `fj`, and the script pack | Owner, 2026-11-23 |
| Forgejo, Gitea, and GitHub work the same. The forge comes from the git remotes | Owner, 2026-11-23 |
| Vale and lexi run in process, through two Go facades that other sessions build | Owner, 2026-11-23 |
| Forgejo is canonical. GitHub `matthewbaggett/vcs` is a push mirror | Owner, 2026-11-23 |

## Open items

The issue list is the work list. See the build order in [the plan](Plans/vcs-go-cli.md).
