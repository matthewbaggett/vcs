# Git verb differences

Issue 75. The git verbs of the binary match the scripts `vcs-branch.sh`, `vcs-commit.sh`,
`vcs-merge.sh`, `vcs-push.sh`, and `vcs-rebase.sh` of agent-standards. This page lists each
difference that stays, and why.

## Matched

| Check | Script | Binary |
|---|---|---|
| Rebase reads the remote branch | Local ref `refs/remotes/origin/<branch>` | The same ref. No network call |
| Rename asks the server | Local ref, then `git ls-remote` | The same two steps |
| Wiring check | `check-wiring.sh --ci` from its own tools tree | Same file, from the submodule `hooks/` or the `hooks/` of the installer layout |
| Check lines | `vcs: check <name>: ok` or `skipped (...)` on stdout | The same lines on stdout |
| Host note | `vcs: host=<forge>` on stderr | The same note on stderr |
| Usage error | Exit 64 | Exit 64 |
| Failed check | Exit 1 | Exit 1 |

## Kept differences

| Difference | Why it stays |
|---|---|
| The binary prints its own usage text. The script printed its header. | The usage text of the binary comes from the verb table. One source feeds the help, the docs, and the site. |
| A wrapper exits 69 when the binary is missing. The old script needed no binary. | A wrapper is one call to the binary. A consumer installs `vcs` first. The wrapper header says so. |
| `FORGEJO_TRACE` maps to `VCS_TRACE` in the wrapper. | One trace switch serves the binary and the scripts. |

## Cases

The 145 cases of `vcs-git.test.sh` have Go tests in `pkg/cli`. They run real git in a
temporary clone. A script test goes away when the same case passes on the binary.
