# Concepts

## Drivers

A driver talks to one kind of forge. `vcs` has two drivers.

| Driver | Forges | Token variable |
|---|---|---|
| Forgejo | Forgejo and Gitea | `VCS_TOKEN`, then the tea login for the host |
| GitHub | github.com | `GITHUB_TOKEN`, then `GH_TOKEN` |

Forgejo and Gitea share one driver, because their web interfaces match. GitHub Enterprise has no
driver yet.

## Forge detection

`vcs` asks the host once what forge it runs. It saves the answer in the config directory, in a file
named `forge-<host>`. Later runs read the file and make no extra call.

## Optional calls

A driver holds a core set of calls. Some calls are optional: edit a pull request, merge, edit an
issue, manage labels, manage milestones, track time, and rename a branch. A driver that lacks one
makes the verb exit 69. The hint then tells the agent to stop and ask.

## Agents and people

`vcs` treats a caller as an agent when you pass `--agent`, when you export `VCS_CALLER=agent`, or when it sees
`CLAUDECODE=1` or `CLAUDE_CODE_ENTRYPOINT`. An agent gets one line on stderr after each run:

```text
vcs: next: <hint>
```

A person at a terminal never sees that line. Use `--human` to force it off.

## Text checks

Each verb that sends text runs the prose lint first. A failed lint exits 65 and sends nothing. The
verb also refuses text that holds an AI credit line.

## Sessions

Some actions have no token API, such as the project board. A web session covers them. Run
`vcs session login`, or `vcs session import`. A verb that needs a session and finds none exits 77.

## Output

Each verb prints plain text that a script can read. The page of each verb names its output. See
[Commands](commands.md).
