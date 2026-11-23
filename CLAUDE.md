@.claude/agent-standards/CLAUDE.md
@.claude/agent-standards/docs/Topics/VCS-Digest.md
@.claude/agent-standards/docs/Topics/Tooling.md

# vcs

One Go command for Forgejo, Gitea, and GitHub. The plan is
[docs/Plans/vcs-go-cli.md](docs/Plans/vcs-go-cli.md). Read it before you build.

Forgejo at `git.grey.ooo/packages/vcs` is canonical. GitHub at `matthewbaggett/vcs` is a
push mirror. Never push to GitHub.

## Working agreement mode

This repo runs in **pull request mode**. Commit and push the task's own branch without
asking. Open a draft pull request at the first push. Mark it ready when CI is green and the
scope is complete. Never merge without the owner's word.

## Stack

| Item | Value |
|---|---|
| Language | Go, latest stable |
| Module path | `git.grey.ooo/packages/vcs` |
| Layout | `cmd/vcs` for the binary, `pkg/` for libraries, `docs/` for guides, `site/` for the built site |
| Build | `go build ./cmd/vcs` |
| Run | `go run ./cmd/vcs <noun> <verb>` |

## Quality gate

Run all of these before each commit. Chain them and read the exit code.

| Check | Command |
|---|---|
| Format | `gofmt -l .` prints nothing |
| Vet | `go vet ./...` |
| Test | `go test ./...` |
| Coverage | `.claude/check-coverage.test.sh && .claude/check-coverage.sh` reaches the 75% floor |
| Lint prose | `.claude/agent-standards/bin/lint-prose-quality.php --base origin/main` |
| Wiring | `.claude/check-wiring.sh --ci` |

The project has no Go code yet. A check that has nothing to run passes. State which checks
you skipped, and why.

## Build order

Follow the build order in the plan. Each step is one issue with its own pull request.
