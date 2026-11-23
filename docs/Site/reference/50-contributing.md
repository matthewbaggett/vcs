# Contributing

## Where

Open issues and pull requests on the canonical repository. Never push to the GitHub mirror.

## Branches

Name a branch `type/short-description`. The types are feature, fix, docs, chore, refactor, test, ci,
perf, build, and revert. Never commit to `main`.

## The gate

Run each check before a commit.

| Check | Command |
|---|---|
| Format | `gofmt -l .` prints nothing |
| Vet | `go vet ./...` |
| Test | `go test ./...` |
| Coverage | `.claude/check-coverage.sh` reaches 75% |
| Prose | `.claude/agent-standards/bin/lint-prose-quality.php --base origin/main` |
| Wiring | `.claude/check-wiring.sh --ci` |

## Generated files

Two folders hold generated files. Never edit them by hand.

| Folder | Rebuild with |
|---|---|
| `docs/Commands/` | `go run ./cmd/vcs docs gen docs/Commands` |
| `site/` | `go run ./cmd/vcs site gen docs/Site site` |

A test fails when either folder drifts from the command tree.

## Prose

Write every page in ASD-STE100 Simplified Technical English. Use short sentences, active voice, and
plain words.
