// Command vcs is one command for Forgejo, Gitea, and GitHub.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.grey.ooo/grey.ooo/lexi"
	"git.grey.ooo/packages/vcs/pkg/cli"
	"git.grey.ooo/packages/vcs/pkg/forge"
	"git.grey.ooo/packages/vcs/pkg/forge/forgejo"
	"git.grey.ooo/packages/vcs/pkg/forge/github"
	"git.grey.ooo/packages/vcs/pkg/lint"
	"git.grey.ooo/packages/vcs/pkg/session"
	"github.com/vale-cli/vale/v3/pkg/vale"
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

// detectKind finds the forge of a host. It keeps the answer for each host in the config
// directory, so only the first run asks the network.
func detectKind(host string) (forge.Kind, error) {
	cache := filepath.Join(configDir(), "forge-"+strings.NewReplacer("/", "_", ":", "_").Replace(host))
	if b, err := os.ReadFile(cache); err == nil {
		switch k := forge.Kind(strings.TrimSpace(string(b))); k {
		case forge.KindForgejo, forge.KindGitea, forge.KindGitHub, forge.KindGitHubEnterprise:
			return k, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	k, err := forge.Detect(ctx, nil, host)
	if err != nil {
		return "", err
	}
	if os.MkdirAll(filepath.Dir(cache), 0o700) == nil {
		_ = os.WriteFile(cache, []byte(string(k)+"\n"), 0o600)
	}
	return k, nil
}

func main() {
	env := cli.Env{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
		ConfigDir:  configDir(),
		Getenv:     os.Getenv,
		IsTerminal: isTTY(os.Stdin) && isTTY(os.Stdout),
		Exec: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
			if err := cmd.Run(); err != nil {
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					return ee.ExitCode()
				}
				fmt.Fprintf(stderr, "vcs: %v\n", err)
				return 1
			}
			return 0
		},
		RemoteURL: func() (string, error) {
			out, err := exec.Command("git", "remote", "get-url", "origin").Output()
			return strings.TrimSpace(string(out)), err
		},
		NewClient: func(host, repo string) (forge.Client, error) {
			kind, err := detectKind(host)
			if err != nil {
				fmt.Fprintf(os.Stderr, "vcs: %v\n", err)
				return nil, forge.ErrUnsupported
			}
			// Forgejo and Gitea share one driver. GitHub has its own. GitHub Enterprise has none yet.
			switch kind {
			case forge.KindGitHub:
				return github.New(github.Config{Repo: repo, Token: github.TokenFromEnv(os.Getenv)}), nil
			case forge.KindForgejo, forge.KindGitea:
			default:
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
	// lexi reads the .alexrc of the working directory, as alex does.
	wd, _ := os.Getwd()
	opts, err := lexi.LoadOptions(wd)
	if err == nil {
		var x *lint.Lexi
		if x, err = lint.NewLexi(opts); err == nil {
			env.Checkers = append(env.Checkers, x)
		}
	}
	// Vale runs only when the working directory holds a .vale.ini.
	if err == nil {
		if _, serr := os.Stat(filepath.Join(wd, ".vale.ini")); serr == nil {
			var v *lint.Vale
			if v, err = lint.NewVale(vale.Options{ConfigPath: filepath.Join(wd, ".vale.ini")}); err == nil {
				env.Checkers = append(env.Checkers, v)
			}
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "vcs: the prose linter cannot start: %v\n", err)
		os.Exit(cli.ExitUnavailable)
	}
	os.Exit(cli.Run(context.Background(), os.Args[1:], env))
}
