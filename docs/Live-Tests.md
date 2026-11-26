# Live tests

The unit tests fake the forge. They check what `vcs` sends and prints. They can't check how a
real server answers. The live suite in `live/` can. It builds the binary and runs it on a
real repository.

## What it covers

| Test | Verbs |
|---|---|
| `TestHelpOfEveryVerb` | `vcs help` for each page in `docs/Commands` |
| `TestIssues` | `issue create/view/edit/list/top/newer/timeline`, `comment add/edit/delete/list`, `time add/list/total`, `attach`, `milestone create/list/view`, `label create/list/rename` |
| `TestGuards` | The lexi check, the credit guard, a usage error, a missing issue |
| `TestGitAndPullRequests` | `git branch/commit/push/merge`, `pr create/ready/draft/edit/status/statuses/bind/tick/heal/merge/close` |
| `TestCI` | `workflow dispatch`, `ci runs/jobs/log/artifacts/rerun/failed`, `pr watch` |
| `TestSecretsAndPackages` | `secret set/list/delete`, `package upload/list/delete` |
| `TestReadOnlyVerbs` | `protection list`, `board list`, `stopwatch list`, `session status`, `runners list` |

Not covered: `session login`, `session import`, `session logout`, `board add`, `board move`,
`pr rebase`, and the GitHub write verbs.

## Run it

The suite runs only with the `live` build tag and a token. Without `VCS_TOKEN` it skips.

```bash
VCS_TOKEN=<token> go test -tags live ./live/ -count=1 -timeout 15m -v
```

| Variable | Default | Meaning |
|---|---|---|
| `VCS_TOKEN` | none | A token that can write to the test repository |
| `VCS_LIVE_HOST` | `git.grey.ooo` | The forge host |
| `VCS_LIVE_REPO` | `geusebio/vcs-scratch` | The throwaway repository |

Never point `VCS_LIVE_REPO` at a real project. The suite opens and merges pull requests, and it
pushes to `main` of that repository.

## How it works

- `TestMain` builds `cmd/vcs` and clones the repository into a temporary folder. It deletes the
  folder at the end.
- The suite runs the binary from the clone, which has no `.vale.ini`. Only the lexi check and the
  credit guard run on the text it sends.
- Each run adds a time stamp to every name it creates, so two runs never collide.
- The first run of `TestCI` lands `.forgejo/workflows/live.yml` on `main` of the repository. The
  runs poll the real runner, so a full run takes about two minutes.
- Cleanup: the suite closes its issues, merges or closes its pull requests, deletes its
  branches, and deletes its secrets and packages. The forge has no delete verb for a label or a
  milestone, so those stay. Their names carry the time stamp.

## When a live test fails

A live failure means the real server did something that the fake server didn't. Fix the verb.
Then update the fake so it matches the real answer. Don't weaken the live test.
