# Artifacts

A CI run can keep files. `vcs` lists them and downloads them. It also attaches a file to an issue or
a pull request.

## List and download

```bash
vcs ci artifacts <run-id>
vcs ci artifact get <run-id> <name> -o out.zip
vcs ci artifact get <run-id> --all -o out-dir
```

A run-id is the number of a run, or the web address of a run.

## Attach a file

```bash
vcs attach <n> <file>
```

The file lands on issue or pull request `<n>`. The command prints the address of the attachment.
Use it for a screenshot in a pull request body.

## Limits

Forgejo supports `actions/upload-artifact@v3` only. A workflow that targets Forgejo must pin that
version.

See [vcs ci artifacts](commands/ci-artifacts.md), [vcs ci artifact get](commands/ci-artifact-get.md),
and [vcs attach](commands/attach.md).
