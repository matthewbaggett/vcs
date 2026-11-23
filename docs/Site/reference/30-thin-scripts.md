# Thin scripts

The agent-standards repository holds shell scripts that wrap forge calls. Each script becomes one
call to the `vcs` binary. A script goes away when the same test passes on the binary.

## Rules

- A thin script has one line: `exec vcs <noun> <verb> "$@"`, plus an exit-code map when the old
  codes differ.
- A script keeps its old name and its old output until no consumer calls it.
- A script test stays until the binary has a Go test for the same case.
- Never switch a script before its verb has a live proof on a real Forgejo server.

## Where it stands

The verbs for pull requests, issues, labels, milestones, time, CI, and the git steps exist. The
scripts of agent-standards switch in a separate pull request in that repository.

The full map lives in the repository, in `docs/Plans/thin-scripts.md`.
