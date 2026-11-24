# vcs

[![Release](https://img.shields.io/github/v/release/matthewbaggett/vcs?sort=semver)](https://github.com/matthewbaggett/vcs/releases/latest)

One Go command for Forgejo, Gitea, and GitHub. Read and change issues, pull requests, CI runs,
labels, milestones, boards, and time from one place, with the same exit codes on each forge.

`vcs` is one static binary. It has no server and no runtime. It checks each text it sends. After
each run it prints a short hint on stderr for an agent, so the agent knows what to do next.

## Install

Each release holds six binaries and one checksum file. Get them from [the latest
release](https://github.com/matthewbaggett/vcs/releases/latest).

| System | Chip | File |
|---|---|---|
| Linux | x86-64 | `vcs_<version>_linux_amd64` |
| Linux | Arm | `vcs_<version>_linux_arm64` |
| macOS | Intel | `vcs_<version>_darwin_amd64` |
| macOS | Apple silicon | `vcs_<version>_darwin_arm64` |
| Windows | x86-64 | `vcs_<version>_windows_amd64.exe` |
| Windows | Arm | `vcs_<version>_windows_arm64.exe` |

1. Download the binary for your system and `vcs_<version>_checksums.txt`.
2. Check the sum: `sha256sum --check --ignore-missing vcs_<version>_checksums.txt`. On macOS, use
   `shasum -a 256 -c` on the same file.
3. Run `chmod +x` on the file. Move it onto your `PATH` and name it `vcs`.
4. Run `vcs --version` to see the version.

To build it yourself, install the latest stable Go and run `go build ./cmd/vcs` in a checkout.

## Quick start

Make a token on your forge. Export it as `VCS_TOKEN` for Forgejo and Gitea, or as `GITHUB_TOKEN`
for GitHub. Then run `vcs` inside a checkout. It reads the `origin` remote to find the host and the
repository. Pass `--host` and `--repo` to name them yourself.

```bash
vcs issue view 8                    # one issue, with its body and comments
vcs pr list --state all --limit 5   # the newest pull requests
vcs pr watch 12                     # wait for the checks of one pull request
vcs ci failed 12                    # the failed jobs of that pull request
vcs ci log <job-id> --tail 50       # the end of one job log
vcs board list                      # the project boards
```

A full change takes four steps:

```bash
vcs git branch create feature/my-change origin/main
vcs git commit -m "Add the thing"
vcs git push origin feature/my-change
vcs pr create "Add the thing" --head feature/my-change --body-file body.md --draft
```

## Commands

Every command has the form `vcs <noun> <verb> [arguments]`.

| Noun | Verbs |
|---|---|
| `pr` | `create`, `draft`, `ready`, `close`, `edit`, `merge`, `rebase`, `heal`, `tick`, `watch`, `list`, `status`, `statuses` |
| `issue` | `create`, `edit`, `list`, `view`, `timeline` |
| `comment` | `add`, `list`, `edit`, `delete` |
| `ci` | `runs`, `jobs`, `log`, `failed`, `artifacts`, `artifact get`, `rerun` |
| `label` | `list`, `create`, `rename` |
| `milestone` | `list`, `view`, `create` |
| `board` | `list`, `view`, `add`, `move` |
| `time` | `add`, `list`, `total` |
| `stopwatch` | `list` |
| `git` | `branch`, `commit`, `push`, `merge`, `rebase` |
| `attach` | upload files to an issue or a pull request |
| `session` | `login`, `import`, `status`, `logout` |
| `version`, `docs gen`, `site gen` | print the version, and build the generated pages |

Each verb has its own page in [docs/Commands](docs/Commands). Run a verb with no argument to see
its usage line.

## Forges

| Forge | Driver | Token variable |
|---|---|---|
| Forgejo | Forgejo | `VCS_TOKEN`, then the tea login for the host |
| Gitea | Forgejo | `VCS_TOKEN`, then the tea login for the host |
| GitHub | GitHub | `GITHUB_TOKEN`, then `GH_TOKEN` |

Forgejo and Gitea share one driver, because their web interfaces match. `vcs` asks the host once
what forge it runs, and it saves the answer. A driver that lacks a call makes the verb exit 69.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | The command did its work. |
| 1 | The command ran and failed, or a check of a git verb failed. |
| 2 | `stopwatch list --check` found a running stopwatch. It means found, not failed. |
| 64 | Usage error. The arguments are wrong. |
| 65 | The text you gave fails the prose lint. `vcs` sends nothing. |
| 69 | No usable driver, or the driver doesn't have the call. Stop and ask. |
| 77 | The verb needs a web session, and there is none. Run `vcs session login`. |

`vcs pr watch` has its own codes. Code 0 means green. Code 1 means a failed check. Code 2 means a
conflict. Code 3 means a closed or merged pull request. Code 4 means a timeout. Code 5 means a
second watcher runs.

## Read more

| Page | What it holds |
|---|---|
| [Site](site/index.html) | The overview, install steps, and quick start |
| [Commands](site/commands.html) | One page for each verb |
| [Exit codes](site/exit-codes.html) | The exit codes and the agent hints |
| [Configuration](site/configuration.html) | Each variable and the config directory |
| [Concepts](site/concepts.html) | The drivers, forge detection, and optional calls |
| [The plan](docs/Plans/vcs-go-cli.md) | Why the project exists, and the build order |
| [Thin scripts](docs/Plans/thin-scripts.md) | The state of the move from shell scripts to the binary |
| [Git verb differences](docs/Plans/git-verb-differences.md) | How the git verbs differ from plain git |

## Contributing

Open issues and pull requests on the canonical repository. The GitHub repository is a read-only
mirror, so never push to it.

- Name a branch `type/short-description`. Never commit to `main`.
- Run the gate before each commit: `gofmt -l .` prints nothing, `go vet ./...`, `go test ./...`,
  `.claude/check-coverage.sh` reaches 75%, and `.claude/check-wiring.sh --ci`.
- Generated files stay generated. Rebuild them with `go run ./cmd/vcs docs gen docs/Commands` and
  `go run ./cmd/vcs site gen docs/Site site`.
- Write every Markdown file in ASD-STE100 Simplified Technical English. See [the contributing
  page](site/contributing.html).
