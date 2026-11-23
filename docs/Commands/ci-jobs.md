# vcs ci jobs

List the jobs of one CI run.

This page is generated from the command tree. Do not edit it by hand.

## Usage

```text
vcs ci jobs <run-id|run-address>
```

## Output

One row for each job, tab-separated: job-id, name, status, attempt, note. The note is `no-log` on a wrapper job. When the run made artifacts, one row follows for each artifact: `artifact`, artifact-id, name, uncompressed size in bytes, note. The note is `expired` for an artifact that the server removed. The server ties an artifact to the run, not to a job, so these rows come after all the job rows.

## Exit codes

- 0 printed the rows
- 1 the call failed or the run is not found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the run-id again with vcs ci runs |
| ok | for a failed job, run vcs ci log <job-id> --tail 80. For an artifact row, run vcs ci artifact get <run-id> <name> |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a run-id or the web address of a run |
