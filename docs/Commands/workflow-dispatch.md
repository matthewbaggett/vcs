# The `vcs workflow dispatch` command

Start one run of a workflow that has a workflow_dispatch trigger. The server gives no run id. Find the run with vcs ci runs. The caller decides whether a manual run is safe.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs workflow dispatch <file> --ref <branch|tag>
```

## Output

ok.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs workflow dispatch nightly.yml --ref main
ok
```

## Flags

| Flag | Meaning |
|---|---|
| `--ref <branch|tag>` | The branch or tag to run the workflow at. Required. |

## Exit codes

- 0 the server accepted the dispatch
- 1 the call failed, or the workflow or the ref is missing
- 64 usage error
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the file name, the ref, and that the workflow has a workflow_dispatch trigger |
| nosession | stop and ask the user for a token with the write:repository scope |
| ok | find the run with vcs ci runs |
| unavailable | stop and ask the user. This host has no workflow dispatch yet |
| usage | give a file name such as nightly.yml and a --ref of a branch or tag |
