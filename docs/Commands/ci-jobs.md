# The `vcs ci jobs` command

List the jobs of one CI run.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs ci jobs <run-id|run-address>
```

## Output

One row for each job, tab-separated: job-id, name, status, attempt, note. The note is `no-log` on a wrapper job.

## Exit codes

- 0 printed the rows
- 1 the call failed or the run isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the run-id again with vcs ci runs |
| ok | for a failed job, run vcs ci log <job-id> --tail 80 |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a run-id or the web address of a run |
