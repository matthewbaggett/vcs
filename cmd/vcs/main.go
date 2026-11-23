// Command vcs is one command for Forgejo, Gitea, and GitHub.
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/cli"
	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/forge/forgejo"
	"git.grey.ooo/packages/vcs/pkg/session"
)

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func configDir() string {
	if d := os.Getenv("VCS_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserConfigDir()
	if err != nil {
		return ".vcs"
	}
	return filepath.Join(home, "vcs")
}

func main() {
	env := cli.Env{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
		ConfigDir:  configDir(),
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
			// A missing session is not an error here. A verb that needs one says so.
			cfg.Cookie, _ = session.Load(configDir(), host)
			return forgejo.New(cfg), nil
		},
	}
	os.Exit(cli.Run(context.Background(), os.Args[1:], env))
}
