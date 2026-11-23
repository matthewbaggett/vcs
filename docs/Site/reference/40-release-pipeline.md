# Release pipeline

The workflow `.forgejo/workflows/build.yml` builds the binaries and publishes them.

## Triggers

| Event | Result |
|---|---|
| A push to `main` | A pre-release named `dev-<short sha>` |
| A push of a tag `v*` | A release with the tag name |

A tag also makes a GitHub release, when the repository holds the secret `GH_RELEASE_TOKEN`.

## Flow

<figure>
<svg viewBox="0 0 640 170" role="img" aria-label="A push to main or a tag v star starts the build. The build makes a pre-release or a release." style="max-width:100%;height:auto" font-family="sans-serif" font-size="14" fill="currentColor">
<defs><marker id="a" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="currentColor"/></marker></defs>
<g fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#a)"><path d="M150 40L220 75"/><path d="M150 130L220 95"/><path d="M360 75L430 40"/><path d="M360 95L430 130"/></g>
<g fill="none" stroke="currentColor" stroke-width="1.5"><rect x="10" y="20" width="140" height="40" rx="6"/><rect x="10" y="110" width="140" height="40" rx="6"/><rect x="220" y="65" width="140" height="40" rx="6"/><rect x="430" y="20" width="200" height="40" rx="6"/><rect x="430" y="110" width="200" height="40" rx="6"/></g>
<g text-anchor="middle"><text x="80" y="45">Push to main</text><text x="80" y="135">Tag v*</text><text x="290" y="90">Build</text><text x="530" y="45">Pre-release dev-sha</text><text x="530" y="135">Release with the tag</text></g>
</svg>
<figcaption>A push to main makes a pre-release. A tag makes a release.</figcaption>
</figure>

## Jobs

<figure>
<svg viewBox="0 0 640 150" role="img" aria-label="The build job feeds the publish job. On a tag, the build job also feeds the GitHub release job." style="max-width:100%;height:auto" font-family="sans-serif" font-size="14" fill="currentColor">
<defs><marker id="b" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="currentColor"/></marker></defs>
<g fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#b)"><path d="M150 75L260 40"/><path d="M150 75L260 110"/></g>
<g fill="none" stroke="currentColor" stroke-width="1.5"><rect x="10" y="55" width="140" height="40" rx="6"/><rect x="260" y="20" width="200" height="40" rx="6"/><rect x="260" y="90" width="200" height="40" rx="6"/></g>
<g text-anchor="middle"><text x="80" y="80">Build the binaries</text><text x="360" y="45">Publish the release</text><text x="360" y="115">Publish to GitHub (tag)</text></g>
</svg>
<figcaption>The build job feeds both publish jobs.</figcaption>
</figure>

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
