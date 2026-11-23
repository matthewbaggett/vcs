# Release pipeline

The workflow `.forgejo/workflows/build.yml` builds the binaries and publishes them.

## Triggers

| Event | Result |
|---|---|
| A push to `main` | A pre-release named `dev-<short sha>` |
| A push of a tag `v*` | A release with the tag name |

A tag also makes a GitHub release, when the repository holds the secret `GH_RELEASE_TOKEN`.

## What it builds

Six binaries: Linux, macOS, and Windows, each on x86-64 and Arm. It also writes
`vcs_<version>_checksums.txt` with the sha256 sum of each file.

The build runs with `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w -X main.version=<version>"`.
Each job runs in `ghcr.io/matthewbaggett/act-runner:php8.5-tools`.

## Quality gate

The workflow `.forgejo/workflows/gate.yml` runs the gate on each pull request and each push to
`main`. The gate runs the format check, vet, the tests, the coverage floor of 75%, the prose lint,
and the wiring check.

## Cut a release

The owner pushes a tag such as `v0.3.0`. The pipeline does the rest.
