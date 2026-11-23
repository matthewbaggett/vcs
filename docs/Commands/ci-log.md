# vcs ci log

Print the log of one CI job, as the runner wrote it.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs ci log <job-id> [--tail <n>]
```

## Output

The plain log text. The command strips terminal colour codes and nothing else.

## Flags

| Flag | Meaning |
|---|---|
| `--tail` | Keep the last <n> lines. |

## Exit codes

- 0 printed the log
- 1 the call failed, the job is not found, or the job has no log
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | a wrapper job has no log. Read the log of the called job, which has a similar name |
| ok | find the first error line, fix the cause, push, then run vcs pr status <n> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give the job-id from vcs ci jobs |
