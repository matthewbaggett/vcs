# The `vcs protection list` command

List the branch protection rules of the repository. With a branch, keep the rules that cover it. A merge that the server refuses with a green run points at this list.

The command tree generates this page. Don't edit it by hand.

## Usage

```text
vcs protection list [<branch>]
```

## Output

One row for each rule: rule, required contexts (a comma list), required approvals, push (none, whitelist, or all), admins (yes or no). No row means the branch has no protection.

## Example

The output below is an example with made-up data, not a real run.

```text
$ vcs protection list main
main	CI / test (pull_request)	1	none	yes
```

## Exit codes

- 0 listed, or no rule covers the branch
- 1 the call failed
- 64 usage error
- 69 no usable driver
- 77 the server refused the token

## Agent hints

A person never sees these. An agent sees one as a `vcs: next:` line on stderr.

| Outcome | Hint |
|---|---|
| fail | check the host, the repository, and the token |
| nosession | stop and ask the user for a token. The list needs one, even on a public repository |
| ok | none |
| unavailable | stop and ask the user. This host has no protection list yet |
| usage | give at most one branch name |
