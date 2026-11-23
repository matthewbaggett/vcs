# Configuration

`vcs` reads its settings from flags and from environment variables. It has no config file.

## Global flags

| Flag | Effect |
|---|---|
| `--agent` | Treat the caller as an agent. Print the hint line |
| `--human` | Treat the caller as a person. Print no hint line |
| `--repo <owner/name>` | Use this repository, not the one of `origin` |
| `--host <host>` | Use this forge host, not the one of `origin` |

## Environment variables

| Variable | Used for | Default |
|---|---|---|
| `VCS_TOKEN` | The token for Forgejo and Gitea | none |
| `GITHUB_TOKEN` | The token for GitHub. It wins over `GH_TOKEN` | none |
| `GH_TOKEN` | The token for GitHub, when you give no `GITHUB_TOKEN` | none |
| `VCS_USER` | The user name for `vcs session login` | none |
| `VCS_PASSWORD` | The password for `vcs session login` | none |
| `VCS_CONFIG_DIR` | Where `vcs` keeps sessions and the forge cache | `os.UserConfigDir()/vcs` |
| `VCS_HEAL_LABEL` | The label of `vcs pr heal` | `➕healing` |
| `VCS_CALLER` | Set to `agent` or `human` | detected |
| `CLAUDECODE` | When `1`, the caller is an agent | none |
| `CLAUDE_CODE_ENTRYPOINT` | When set, the caller is an agent | none |
| `NETRC` | The netrc file to read when you give no token | `~/.netrc` |

Never put a token in a command line or in a commit. Set it in the environment.
