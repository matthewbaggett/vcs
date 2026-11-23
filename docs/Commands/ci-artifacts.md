# The `vcs ci artifacts` command

List the artifacts of one CI run.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs ci artifacts <run-id|run-address>
```

## Output

One row for each artifact, tab-separated: artifact-id, name, uncompressed size in bytes, note. The note is `expired` for an artifact that the server removed. An empty list isn't an error.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs ci artifacts 331
77	coverage	20480	
78	binaries	5242880	expired
```

## Exit codes

- 0 printed the rows
- 1 the call failed or the run isn't found
- 64 usage error
- 69 no usable driver, or the host has no artifact route

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | get the run-id again with vcs ci runs |
| ok | run vcs ci artifact get <run-id> <name> to download one |
| unavailable | stop and ask the user. This host has no artifact route yet |
| usage | give a run-id or the web address of a run |
