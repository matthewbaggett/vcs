# Thin scripts plan

Build order step 6 of [the plan](vcs-go-cli.md). Issue 8. Each `vcs-*.sh` script becomes one
call to the `vcs` binary. A script goes away when the same test passes on the binary.

## Where the scripts live

The scripts live in the agent-standards repository, in `tools/forgejo/`. This repository holds
only links to them in `bin/`. So the change happens in agent-standards, in a separate pull
request. It reaches every project that imports that repository.

## Rules

- A thin script has one line: `exec vcs <noun> <verb> "$@"`, plus an exit-code map if the old
  codes differ.
- A script keeps its old name and its old output until a consumer stops calling it.
- A script test stays until the binary has a Go test for the same case. Then the test goes with
  the script.
- Never switch a script before its verb has a live proof on a real Forgejo server.

## Map

| Script | Verb | State |
|---|---|---|
| `vcs-pr-status.sh` | `vcs pr status` | Done. The script runs the binary |
| `vcs-pr.sh create`, `ready` | `vcs pr create`, `vcs pr ready` | Done. The script runs the binary |
| `vcs-pr.sh edit`, `merge` | `vcs pr edit`, `vcs pr merge` | Done. The script runs the binary |
| `vcs-pr.sh` close | `vcs pr close` | Done. The script runs the binary |
| `vcs-pr.sh` comment | `vcs comment add` | Done. The script runs the binary |
| `vcs-pr-list.sh` | `vcs pr list` | Done. The script runs the binary |
| `watch-pr.sh` | `vcs pr watch` | Done. The script runs the binary since v0.0.6 |
| `vcs-issue.sh create` | `vcs issue create` | Done. The script runs the binary |
| `vcs-issue.sh edit` | `vcs issue edit` | Done. The script runs the binary |
| `vcs-issue.sh` label, close | `vcs issue edit` with `--add-label`, `--rm-label`, `--state closed` | Done. The script runs the binary |
| `vcs-issue-list.sh`, `vcs-issue-view.sh` | `vcs issue list`, `view` | Done. The script runs the binary |
| `vcs-issue-new.sh` | `vcs issue newer`, `vcs issue top` | Done. The script runs the binary since v0.0.8, and keeps its old path for an older binary |
| `vcs-issue-timeline.sh` | `vcs issue timeline` | Done. The script runs the binary since v0.0.7, and keeps its old path for an older binary |
| `vcs-comment.sh` | `vcs comment add` | Done. The script runs the binary |
| `vcs-board.sh` | `vcs board list`, `view`, `add`, `move` | Needs a live proof with a session |
| `vcs-ci-runs.sh`, `vcs-ci-jobs.sh`, `vcs-ci-log.sh` | `vcs ci runs`, `jobs`, `log` | Done. The script runs the binary |
| `vcs-ci-failed.sh` | `vcs ci failed` | Done. The script runs the binary |
| `vcs-attach.sh` | `vcs attach` | Needs a live proof with a token |
| `vcs-session.sh` | `vcs session` | Needs a live proof with a login |
| `vcs-statuses.sh` | `vcs pr statuses` | Done. The script runs the binary since v0.0.7, and keeps its old path for an older binary |
| `vcs-label.sh`, `vcs-milestone.sh`, `vcs-time.sh` | `vcs label`, `milestone`, `time` | Done. The script runs the binary |
| `vcs-stopwatch.sh` | `vcs stopwatch list` | Ready to switch. The script has only the list action |
| `vcs-secret.sh`, `vcs-package.sh`, `vcs-protection.sh`, `vcs-runners.sh`, `vcs-workflow.sh`, `vcs-pr-bind.sh` | none | Gap. The binary has no verb for them |
| `vcs-tick-merge.sh`, `vcs-heal-label.sh` | `vcs pr tick`, `vcs pr heal` | Tick: ready to switch. Heal: needs a live proof of its write paths |
| `vcs-branch.sh`, `vcs-commit.sh`, `vcs-push.sh`, `vcs-merge.sh`, `vcs-rebase.sh` | `vcs git ...` | Built. The wiring retry and the server rename need a live proof |

## Order of work

1. Done. Switch the scripts that have a verb. Several agent-standards pull requests. Each script
   test ran on the binary first. Only `vcs-stopwatch.sh` still waits.
2. Done. Add the verbs for the read gaps: `pr list`, `issue list`, `issue view`, `ci failed`.
3. Done. Add the write gaps: `pr edit`, `pr merge`, `issue edit`, `label`, `milestone`, `time`.
4. Done. Add the git verbs, with the guard checks the hook runs today.
5. Verbs done. The script rewrite waits on a live proof of `pr heal`. Rewrite `vcs-tick-merge.sh` and `vcs-heal-label.sh` on top of the verbs.
6. Delete the old `forgejo-` names in the release that ships the binary.

## Blockers

- The binary needs a release. The build pipeline publishes one only on a tag.
- The agent-standards CI image must hold the binary, or fetch it, before a script calls it.
- A live proof needs a Forgejo account. Only the owner can give one.
