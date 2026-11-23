# Overview

`vcs` is one Go command for Forgejo, Gitea, and GitHub. It reads and writes pull requests, issues,
labels, milestones, boards, time, and CI runs. It also wraps the git steps of a normal change.

The command has no server step and no runtime. It's one static binary.

## Why it exists

A team that works with agents needs one stable command. The command checks each text it sends. It
gives a short hint after each run, so an agent knows what to do next. It exits with the same codes
on every forge.

## Where to start

| You want to | Read |
|---|---|
| Install the binary | [Install](install.md) |
| Run your first commands | [Quickstart](quickstart.md) |
| Learn how the drivers work | [Concepts](concepts.md) |
| Add a token or a variable | [Configuration](configuration.md) |
| Find one verb | [Commands](commands.md) |
| Read the exit codes | [Exit codes and hints](exit-codes.md) |

## Where the code lives

The code is public at `github.com/matthewbaggett/vcs`. That repository is a read-only mirror of the
canonical one.
