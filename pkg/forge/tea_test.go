package forge

import (
	"os"
	"path/filepath"
	"testing"
)

const teaFixture = `logins:
    - name: other
      url: https://forge.example
      token: made-up-other
      default: false
      user: alice
    - name: first
      url: https://git.example.test
      token: made-up-first
      default: false
    - name: chosen
      url: "https://git.example.test"
      token: 'made-up-default'
      default: true
    - name: empty
      url: https://empty.example
      token: ""
preferences:
    editor: false
`

func TestParseTeaToken(t *testing.T) {
	cases := []struct{ host, want string }{
		{"forge.example", "made-up-other"},
		{"git.example.test", "made-up-default"},
		{"GIT.EXAMPLE.TEST", "made-up-default"},
		{"empty.example", ""},
		{"missing.example", ""},
	}
	for _, c := range cases {
		if got := parseTeaToken(teaFixture, c.host); got != c.want {
			t.Errorf("host %q: got %q, want %q", c.host, got, c.want)
		}
	}
	if got := parseTeaToken("", "x"); got != "" {
		t.Errorf("empty config gave %q", got)
	}
}

func TestParseTeaTokenFirstWhenNoDefault(t *testing.T) {
	cfg := "logins:\n  - name: a\n    url: https://h.example\n    token: made-up-a\n  - name: b\n    url: https://h.example\n    token: made-up-b\n"
	if got := parseTeaToken(cfg, "h.example"); got != "made-up-a" {
		t.Errorf("got %q", got)
	}
}

func TestTeaTokenReadsTheConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(teaFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEA_CONFIG", path)
	if got := TeaToken("git.example.test"); got != "made-up-default" {
		t.Errorf("got %q", got)
	}
	// The read leaves the file as it was.
	after, _ := os.ReadFile(path)
	if string(after) != teaFixture {
		t.Error("TeaToken changed the config file")
	}
	t.Setenv("TEA_CONFIG", filepath.Join(dir, "missing.yml"))
	if got := TeaToken("git.example.test"); got != "" {
		t.Errorf("missing file gave %q", got)
	}
}

func TestForgejoTokenOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(teaFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEA_CONFIG", path)
	env := map[string]string{}
	get := func(k string) string { return env[k] }
	if got := ForgejoToken(get, "git.example.test"); got != "made-up-default" {
		t.Errorf("tea login: got %q", got)
	}
	env["VCS_TOKEN"] = "made-up-env"
	if got := ForgejoToken(get, "git.example.test"); got != "made-up-env" {
		t.Errorf("VCS_TOKEN must win: got %q", got)
	}
}
