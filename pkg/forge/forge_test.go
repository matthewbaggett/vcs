package forge

import "testing"

func TestParseRemote(t *testing.T) {
	good := map[string][2]string{
		"https://git.grey.ooo/packages/vcs.git":       {"git.grey.ooo", "packages/vcs"},
		"https://user@git.grey.ooo:8443/packages/vcs": {"git.grey.ooo", "packages/vcs"},
		"git@github.com:matthewbaggett/vcs.git":       {"github.com", "matthewbaggett/vcs"},
		"ssh://git@git.grey.ooo/packages/vcs.git":     {"git.grey.ooo", "packages/vcs"},
	}
	for in, want := range good {
		h, r, err := ParseRemote(in)
		if err != nil || h != want[0] || r != want[1] {
			t.Errorf("%s -> %s %s %v", in, h, r, err)
		}
	}
	for _, in := range []string{"", "nonsense", "https://host/onlyone"} {
		if _, _, err := ParseRemote(in); err == nil {
			t.Errorf("%q should fail", in)
		}
	}
}

func TestParseNetrc(t *testing.T) {
	data := "machine a.example login x password y\nmachine git.example\n  login me\n  password secret\n"
	l, p := parseNetrc(data, "git.example")
	if l != "me" || p != "secret" {
		t.Errorf("got %q %q", l, p)
	}
	if l, p := parseNetrc(data, "none.example"); l != "" || p != "" {
		t.Errorf("got %q %q", l, p)
	}
}
