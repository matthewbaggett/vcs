# Quickstart

This page takes you from a new install to a merged change.

## 1. Make a token

Make a token on your forge. Then export it. For Forgejo and Gitea, use `VCS_TOKEN`. For GitHub, use
`GITHUB_TOKEN`.

```bash
export VCS_TOKEN=<your token>
```

See [Configuration](configuration.md) for each variable.

## 2. Work in a checkout

`vcs` reads the `origin` remote of the current directory. It finds the host and the repository from
that remote. To name them yourself, pass `--host` and `--repo`.

## 3. Make a branch and a commit

```bash
vcs git branch create feature/my-change origin/main
vcs git commit -m "Add the thing" -m "Say why in the second message"
vcs git push origin feature/my-change
```

## 4. Open a draft pull request

```bash
vcs pr create "Add the thing" --head feature/my-change --body-file body.md --draft
```

## 5. Watch the checks, then mark it ready

```bash
vcs pr watch 12
vcs pr ready 12
```

`vcs pr watch` exits 0 when every check passes. It exits 1 when one fails. Use it to wait.

## 6. Read what failed

```bash
vcs ci failed 12
vcs ci log <job-id> --tail 50
```

## Next

Read [Concepts](concepts.md), then browse [the command list](commands.md).
