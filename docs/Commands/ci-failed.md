# The `vcs ci failed` command

List the red jobs of a pull request head or a commit, with the first error of each log.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs ci failed <pr-number|sha>
```

## Output

One row for each red job, tab-separated: run-id, job-id, job-name, first error. The first error is the text of the first `::error::` line. The column is blank when the log has none, and `(log unreadable)` when the read failed. A wrapper job has no log and a blank column.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs ci failed 12
331	902	gate	go vet found a problem in pkg/cli/cli.go
```

## Exit codes

- 0 printed the rows (a commit with no red job prints nothing)
- 1 the call failed, the target isn't found, or a run belongs to another commit
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number or the sha, then run the command again |
| ok | read the first-error column before you call a failure a flake. For the whole log, run vcs ci log <job-id> --tail 80 |
| unavailable | stop and ask the user. This host has no driver yet |
| usage | give a pull request number (1 to 6 digits) or a commit sha (7 to 40 hex characters) |
