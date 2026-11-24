# The `vcs issue timeline` command

Print the events of one issue or pull request, oldest first.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs issue timeline <n>
```

## Output

Rows, tab-separated, no header: event id, created, type, user, detail. The type is the one that the server gives, such as comment, label, change_title, pull_push, or commit_ref. The detail of a label is +name or -name. The detail of a title change is old -> new. The detail of a push is push or force-push with the commit ids. Other types print their text on one line. An empty timeline isn't an error.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs issue timeline 14
101	2026-11-23T09:12:00Z	label	matt	+in-progress
102	2026-11-23T09:20:00Z	comment	matt	Started on this.
```

## Exit codes

- 0 printed the rows
- 1 the call failed or the issue isn't found
- 64 usage error
- 69 no usable driver

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the number with vcs issue list, then run the command again |
| ok | compare the time of the last event with the time of your claim, then decide if the claim is stale |
| unavailable | stop and ask the user. This host has no driver for timelines yet |
| usage | run vcs issue timeline <n> with the number of one issue or pull request |
