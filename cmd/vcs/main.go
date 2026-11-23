// Command vcs is one command for Forgejo, Gitea, and GitHub.
package main

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/cli"
	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/forge/forgejo"
)

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func main() {
	env := cli.Env{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Getenv:     os.Getenv,
		IsTerminal: isTTY(os.Stdin) && isTTY(os.Stdout),
		RemoteURL: func() (string, error) {
			out, err := exec.Command("git", "remote", "get-url", "origin").Output()
			return strings.TrimSpace(string(out)), err
		},
		NewClient: func(host, repo string) (forge.Client, error) {
			if host == "github.com" {
				return nil, forge.ErrUnsupported
			}
			cfg := forgejo.Config{BaseURL: "https://" + host, Repo: repo, Token: os.Getenv("VCS_TOKEN")}
			if cfg.Token == "" {
				cfg.Login, cfg.Password = forge.NetrcLogin(host)
			}
			return forgejo.New(cfg), nil
		},
	}
	os.Exit(cli.Run(context.Background(), os.Args[1:], env))
}
