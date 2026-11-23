# Research: A Go command that replaces `tea`, `fj`, and the script pack

This note decides whether the project writes its own forge command in Go. It makes no
code change. Checked on 2026-11-23 with `tea` 0.15.1 and the current `fj`.

## Answer

Yes. Write one Go binary, `vcs`. It talks to Forgejo, Gitea, and GitHub through one
interface. The 15,600 lines of `tools/forgejo/*.sh` are the spec. They already name every gap.

## What hurts today

| Pain | Why `tea` and `fj` fail | How the pack works around it |
|---|---|---|
| Full CI logs | `fj actions` has no log command. A busy server answers a log read with 504 | `forgejo-ci-log.sh` retries three times, one at a time |
| Restart a task | The Forgejo API has no rerun route. The web route needs a session | None. A person presses the button in the browser |
| Attach a screenshot | Neither tool uploads an asset to a PR | `vcs-attach.sh` calls the asset route with `curl` |
| Project boards | Forgejo has no board API. The routes are web only | `forgejo-board.sh` uses a session cookie |
| Tool choice | Each script tries `fj`, then `tea`, then `curl` | One helper, `forgejo_api`, hides the choice |
| Wait on a PR | Neither tool waits on checks and merge state together | `watch-pr.sh` polls |

## Proposed design

One interface, three drivers. A command never names the host.

| Layer | Job |
|---|---|
| `forge.Client` interface | Issue, PR, comment, label, milestone, time, CI run, CI log, asset, board |
| `forgejo` driver | Token calls with a token. Web routes with a session cookie for rerun, logs, and boards |
| `gitea` driver | Same as `forgejo`. Use the `code.gitea.io/sdk/gitea` module. Probe the version, because the web routes differ |
| `github` driver | Token calls and GraphQL. GitHub Projects v2 is GraphQL only |
| `session` package | Cookie jar, login, import of `i_like_gitea`. Port of `forgejo-session.sh` |
| `lint` package | Calls `tools/agent-tools/pkg/prose`. Every text the CLI sends gets linted first |
| `guard` package | The version control rules of `hooks/vcs-guard.sh`: no AI trailer, no commit on `main` |

The host comes from the remotes of the git repo, not from a setting. `--host` overrides it.
The same command works the same on every host. The user never names the forge.

## Remotes and mirrors

`vcs` reads every remote of the git repo, not only `origin`. It detects the forge of each
remote with the table below. It then builds a mirror map.

| Case | What `vcs` does |
|---|---|
| One remote | That remote is the home. Every verb goes there |
| Two remotes, same repo path on two forges | Mirror pair. One remote is canonical, the other is a read-only copy |
| Remotes that differ in repo path | Not a pair. Ask the user once. Save the answer in `.vcs.toml` |

The canonical remote is, in order: `canonical = "<remote>"` in `.vcs.toml`, then the remote
named `forgejo` or `canonical`, then `origin`. A write verb goes to the canonical remote
only. A read verb can ask the mirror too, with `--from mirror`. `vcs mirror status` prints
each pair, which side is canonical, the last sync commit on each side, and the lag in commits.
`vcs mirror set <remote>` changes the canonical side. A write to a read-only mirror exits 64
and names the canonical remote.

Two numbers can differ between a pair: the PR index and the issue index. `vcs` keeps a map of
them in `.vcs/mirror.json`, so `vcs pr view 12` finds the right item on both sides.

## Interactive use

A terminal gets a full-screen interface. A pipe gets plain tab-separated text, as the pack
does now. The mode follows the terminal, and `--plain` or `--tui` forces it.

| Screen | Opens with | Does |
|---|---|---|
| PR list and PR view | `vcs pr` | Filter, open, mark ready, merge, view checks and merge state live |
| CI run view | `vcs ci` | Job tree, log pane that follows, `r` to rerun, `R` to rerun failed jobs |
| Issue board | `vcs board` | Columns, move a card with a key, open an issue |
| Attach | `vcs pr attach` | Pick files, preview, post |
| Mirror status | `vcs mirror` | Both sides, lag, switch the canonical side |

Build it with Bubble Tea, the Go terminal framework. Each screen draws from the same
`forge.Client` calls as the plain commands, so the two modes never differ in what they do.

## Detecting the forge

Detection runs in this order. The first hit wins. `vcs` keeps the result for each host.

| Step | Check | Result |
|---|---|---|
| 1 | `--host` flag, or `.vcs.toml` has `forge = "..."` | Use it |
| 2 | Host is `github.com` | GitHub |
| 3 | `GET /api/v1/version` answers | Forgejo or Gitea. The version string has `+gitea-` for Gitea, and `forgejo` for Forgejo |
| 4 | `GET /api/v3/meta` answers | GitHub Enterprise |
| 5 | Nothing answers | Exit 69 with the hosts tried |

A verb that a host lacks exits 69 with one line that names the host and the verb.

## Commands

| Command | Forgejo and Gitea | GitHub |
|---|---|---|
| `vcs pr create, edit, ready, merge, status` | Token calls | Token calls |
| `vcs pr watch <n>` | Poll checks and merge state. Exit 0 only when both are good | Same |
| `vcs attach <n> <file>...` | `POST /issues/{n}/assets`, then one comment with the image links | No public upload API. Commit the file to a branch, or use a release asset. Link it |
| `vcs ci runs, jobs, failed` | `/actions/runs`, `/actions/tasks` | `/actions/runs` |
| `vcs ci log <job>` | Web route with session. Stream it. Retry on 504 | Zip download of the run logs. Unpack one job |
| `vcs ci rerun <run>` | `POST` the web `rerun` route. `--failed` reruns red jobs only | `POST /actions/runs/{id}/rerun`, and `rerun-failed-jobs` |
| `vcs board list, view, add, move` | Web routes with session | GraphQL Projects v2 |
| `vcs issue`, `comment`, `label`, `milestone`, `time` | Token calls | Token calls. GitHub has no time tracking: print "unsupported" and exit 69 |

## All verbs

`vcs <noun> <verb>`. This table is the union of `tea` 0.15.1, `fj`, and the script pack. A
mark in the tool columns means the tool has the verb today. The last column lists the verbs
that only `vcs` adds. Each verb runs on all three hosts unless a note says otherwise.

| Noun | `tea` verbs | `fj` verbs | Pack verbs | New in `vcs` |
|---|---|---|---|---|
| `issue` | list, create, edit, reopen, close | create, edit, comment, assign, unassign, close, search, view, templates, browse | list, view, new, timeline | `claim`, `unclaim`, `link`, `lint` |
| `pr` | list, checkout, clean, create, close, reopen, edit, review, approve, reject, merge, reply, review-comments, resolve, unresolve | search, create, view, status, checkout, comment, assign, unassign, edit, close, merge, browse, review | status, list, bind, ready, draft, tick-merge | `watch`, `rebase`, `attach`, `conflicts` |
| `comment` | add, list, edit, delete | (under `issue` and `pr`) | list, edit, delete | `release-claim` |
| `label` | list, create, update, delete | (under `repo labels`) | list, create, rename, heal | `sync` |
| `milestone` | list, create, close, delete, reopen, issues | none | list, view, create | `assign` |
| `release` | list, create, delete, edit, assets | create, edit, delete, list, view, browse, asset create, asset delete, asset download | none | `notes` |
| `tag` | none | create, delete, list, view | none | none |
| `repo` | list, search, create, create-from-template, fork, migrate, delete, edit | create, fork, migrate, view, readme, clone, star, unstar, watch, unwatch, delete, browse, labels, edit, units | none | `protect-show` |
| `branch` | list, protect, unprotect, rename | none | protection | `delete-merged` |
| `ci` | runs, workflows | actions tasks, dispatch | runs, jobs, log, failed, workflow dispatch | `rerun`, `rerun --failed`, `cancel`, `watch`, `log --follow` |
| `secret` | secrets (list, create, delete) | secrets (list, create, delete) | list, set, delete | none |
| `variable` | variables | variables (list, create, delete) | none | `set` |
| `time` | list, add, delete, reset | none | log, list, total | `stopwatch start`, `stopwatch stop` |
| `wiki` | list, view, revisions, create, edit, delete | contents, view, clone, browse | none | none |
| `webhook` | list, create, delete, update | none | none | `test` |
| `org` | list, create, delete | list, view, create, edit, activity, members, visibility, team, label, repo | none | none |
| `user` | admin users | search, view, browse, follow, unfollow, following, followers, block, unblock, repos, orgs, activity, edit, key, gpg | none | none |
| `notification` | ls, read, unread, pin, unpin | none | none | none |
| `package` | none | none | upload, list, prune, delete | none |
| `board` | none | none | list, view, add, move | `create`, `column add` |
| `runner` | none | none | list | `token` |
| `protection` | protect, unprotect | none | list | `set` |
| `auth` | login, logout | auth | session login, import, status, logout | `whoami` |
| `api` | api | none | `forgejo_api` helper | `vcs api`, one call for all three hosts |
| `lint` | none | none | none | `check <text>`, `config show` |

Notes:
- GitHub has no `time` and no `stopwatch`. It has no `wiki` write route. Those verbs exit 69
  there.
- `board` and `ci rerun` use web routes on Forgejo and Gitea.
- Build the nouns in the order of the pain, not the order of this table. See the build order.

## Enforcement

`vcs` is the one door to the forge and to git writes. It enforces four things. A check that
fails stops the verb, sends nothing, and exits 65. `vcs` has no `--no-verify` flag.

| Layer | What it enforces | Where it runs |
|---|---|---|
| Git hooks | `vcs hooks install` writes `pre-commit`, `commit-msg`, and `pre-push` that call `vcs hooks run <name>`. `vcs hooks check` fails when a hook is missing or edited. A write verb runs `hooks check` first | `commit`, `push`, `pr create`, and in CI |
| Linters | Vale, lexi or alex, and the project's own gate table, from `.vcs.toml`. `vcs lint run` runs them on the staged files, or on the text of a send | Before each commit and each send |
| Validation | Branch name `type/short-description`. No commit on `main`. No force-push. No AI credit line. Title and body shape of a PR. Required sections of the PR body. No secret in the diff | Each write verb |
| Language filtering | Plain-language rules (ASD-STE100), inclusive language (lexi), and a profanity list with a `profanitySureness` level. A project adds a deny list and a permit list of words | Each text that goes to the forge: issue, PR, comment, commit message |

A rule can be `error`, `warn`, or `off` in `.vcs.toml`. The default is `error`. A rule set to
`off` shows in `vcs lint config show`, so nobody turns a rule off in secret. The user file can
make a rule stricter than the project file. It can't make one weaker.

Each check prints one line, `vcs: check <name>: ok`, `skipped`, or the failure with the
line and the rule. A check that can't run, such as a missing linter binary, exits 69. It
never counts as a pass.

## Messages for automation

`vcs` finds out who called it. When the caller is an agent such as Claude, it adds a second
message that says what to do next. A person never sees that message.

| Signal | Meaning |
|---|---|
| `VCS_CALLER=agent` or `--caller agent` | Forced. `VCS_CALLER=human` forces the other way |
| The env var `CLAUDECODE=1`, or any value in `CLAUDE_CODE_ENTRYPOINT` | Claude Code called it |
| No terminal on stdin and stdout, and no `--plain` | Probably a script or an agent. Treat as agent |
| A terminal on both | A person |

Rules for the agent message:

- It goes on stderr as a line that starts `vcs: next:`. Stdout keeps the data, as now.
- It appears only when it differs from what the human message already says. A hint that
  repeats the human line is noise.
- It names one action with the exact command. Example: after a red check, `vcs: next: run
  vcs ci log 4412 --tail 80, fix the cause, push, then run vcs pr watch 12`.
- It names a stop when a person must decide. Example: `vcs: next: stop and ask the user. The
  merge needs a review.`
- With `--json`, the same text is the `next` field of the result. The field is absent for a
  person.
- It never holds a secret, and it never tells the agent to bypass a check.

Each verb keeps its hints in one table in the code, so a test can check every exit path has
one. The hints live next to the rules in `docs/Topics/VCS.md`, so a rule change updates both.

## Documentation site

The docs build into a static site in a subdirectory, `site/`, of `packages/vcs`.

| Part | Source |
|---|---|
| Guides | The Markdown files in `docs/` |
| Command pages | Generated from the command tree: each verb's usage, flags, exit codes, and hint table |
| Plans and notes | `docs/Plans/`, listed apart from the guides |
| Search | An index built at the same time, with no server |

`vcs docs build` writes `site/`. It needs no Node. The generator is Go code inside the `vcs` binary. It lints each page with the same linters as every other text.

Regeneration is automatic:

1. A CI job builds the site on each push to `main` and commits `site/` only when it changes.
2. The `pre-push` hook runs `vcs docs check`. It fails when `site/` is older than `docs/` or
   than the command tree.
3. A change to a flag or a hint changes the command pages, so a code change can't leave the
   docs behind.

The site publishes from `site/` on the forge's pages feature, or on any static host.

## One portable binary

Another session builds a Go facade for Vale and another for lexi. `vcs` imports both as Go
modules and calls them in process. The binary then needs no Node, no Vale install, and no
`lexi` on the path. It runs on a bare machine and in a `FROM scratch` image.

| Part | Source | How `vcs` uses it |
|---|---|---|
| Vale facade | The Vale session's module | `lint.Vale.Check(text, config) -> []Finding` |
| lexi facade | The lexi session's module | `lint.Lexi.Check(text, config) -> []Finding` |
| ASD-STE100 styles | `tools/vale/styles`, embedded with `go:embed` | Loaded from memory. No `StylesPath` on disk |
| Project styles | A `StylesPath` in `.vcs.toml` | Merged over the embedded set |

Both facades must meet one contract, so `vcs` can swap or test them:

- Take text and a file kind, and return findings with line, column, rule id, and message.
- Take the config as a value. Read no file of their own and print nothing.
- Keep no global state, so `vcs` can lint many texts at once.
- Pin their upstream version, and print it for `vcs version`.

Two checks hold the build to this. A parity test runs the facades and the real Vale and lexi
on the same corpus and compares findings. A size test fails the build when the binary grows
past a set limit.

`vcs lint run` uses the embedded linters by default. `--external` runs the installed `vale`
or `lexi` instead, for a person who tests a new rule.

## Linter configuration

Every text `vcs` sends goes through the prose linters first. The linters need settings that
a project can change. Today the settings sit in `.vale.ini` and `tools/alex/filter.json`.

| Linter | Setting file today | What `vcs` adds |
|---|---|---|
| Vale | `.vale.ini` or `tools/vale/vale.ini` | `lint.vale.config`, a path. `lint.vale.styles`, a list of styles |
| alex | `tools/alex/filter.json` | `lint.alex.config`, a path. `lint.alex.allow`, a list of rule ids to drop |
| lexi | `.alexrc` (JSON or YAML). lexi is a static Go port of alex 11.0.1 and reads the same file | `lint.alex.*` settings apply to lexi as well. `vcs` prefers `lexi` and falls back to `alex` |

The runner image ships `vcs`, which holds both linters. `lexi` also stays available from
`ghcr.io/matthewbaggett/lexi` for a job that wants it alone. Either way, Node leaves the lint path. lexi has gaps: only the JSON reporter, and no `.alexrc.js`. The `vcs` lint layer needs none of them.

One file, `.vcs.toml` in the repo root, holds the settings. The user file
`~/.config/vcs/config.toml` holds the defaults. A project value beats a user value. A flag
beats both. `vcs lint config show` prints the merged result and the file each value came from.
A project that has no setting runs the linters with the defaults of this repo, as now.

## Gaps that stay gaps

- Rerun and boards on Forgejo depend on web routes. A Forgejo upgrade can break them. Pin
  the tested server version in a test, and fail with a clear error on a mismatch.
- A login that needs a two-factor code needs the `import` path.
- GitHub has no time tracking and no stopwatch.

## Build or reuse

| Option | Cost | Verdict |
|---|---|---|
| Keep the scripts | No cost. Each tool probe makes a call slow. 15,600 lines of Bash is hard to test | Stop adding to it |
| Patch `fj` upstream | `fj` is Rust. Upstream decides what lands | File the gaps. Don't wait |
| Fork `tea` | Go, so close to this project's stack. Large, and Gitea first | Borrow its SDK use. Don't fork |
| New Go binary | The `agent-tools` module already has Go, tests, and the prose linter | Choose this one |

## Build order

1. `forge.Client` plus the Forgejo driver for read commands: `pr status`, `ci runs`,
   `ci jobs`, `ci log`. Test each with a recorded server.
2. `ci rerun`, `attach`, and the session package. These fix the three worst pains.
3. `pr watch`, then the write commands with the lint and guard layers.
4. The GitHub driver, read commands first.
5. `board`, on both drivers.
6. Make each `vcs-*.sh` a thin call to the binary. Delete a script when its test passes.

## Open questions

| Open point | Recommendation |
|---|---|
| Where the binary lives | A new module, `tools/vcs`, not `tools/agent-tools`. It ships alone and shares only the `prose` package |
| Gitea or Forgejo only | Both. Share one driver with a version probe |
| The `forgejo-` script names | Drop them in the same release that ships the binary |
