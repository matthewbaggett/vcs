package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

type pkgFake struct {
	fake
	pkgs        []forge.Package
	uploaded    string
	data        string
	linkErr     error
	uploadErr   error
	deleted     string
	linked, did bool
}

func (f *pkgFake) PackageList(_ context.Context, name string) ([]forge.Package, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []forge.Package
	for _, p := range f.pkgs {
		if name == "" || p.Name == name {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *pkgFake) PackageUpload(_ context.Context, name, version, file string, r io.Reader, size int64) error {
	b, _ := io.ReadAll(r)
	f.data, f.uploaded = string(b), name+" "+version+" "+file
	return f.uploadErr
}
func (f *pkgFake) PackageLink(context.Context, string) error { f.linked = true; return f.linkErr }
func (f *pkgFake) PackageDelete(_ context.Context, name, version string) error {
	f.deleted = name + " " + version
	return f.err
}
func (f *pkgFake) PackageURL(name, version, file string) string {
	return "https://h/" + name + "/" + version + "/" + file
}

const helloSum = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"

func TestPackage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tool-1.2.tgz")
	os.WriteFile(file, []byte("hello\n"), 0o644)
	empty := filepath.Join(dir, "empty.tgz")
	os.WriteFile(empty, nil, 0o644)
	bad := filepath.Join(dir, "-bad.tgz")
	os.WriteFile(bad, []byte("x"), 0o644)
	link := filepath.Join(dir, "link.tgz")
	os.Symlink(file, link)
	now := time.Now().UTC()
	ts := func(days int) string { return now.Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339) }
	mk := func() *pkgFake {
		return &pkgFake{pkgs: []forge.Package{
			{Name: "tool", Version: "1.0", Created: ts(90)},
			{Name: "tool", Version: "1.1", Created: ts(5)},
			{Name: "tool", Version: "0.9", Created: ts(40)},
		}}
	}

	// list
	c := mk()
	code, out, errb := run(t, c, nil, false, "package", "list", "tool")
	if code != ExitOK || !strings.Contains(out, "tool\t1.1\t") || !strings.Contains(out, "\tnewest\n") ||
		!strings.Contains(out, "\told\n") || !strings.Contains(errb, "2 older version(s)") {
		t.Fatalf("list: %d %q %q", code, out, errb)
	}
	if code, out, _ := run(t, c, nil, false, "package", "list", "--stale-days", "60"); code != ExitOK || strings.Count(out, "\told\n") != 1 {
		t.Errorf("stale: %d %q", code, out)
	}

	// upload: dry run sends nothing
	c = mk()
	code, out, _ = run(t, c, nil, false, "package", "upload", "tool", "1.2", file, "--dry-run")
	if code != ExitOK || !strings.HasPrefix(out, "dry-run\thttps://h/tool/1.2/tool-1.2.tgz\t"+helloSum+"\n") || c.uploaded != "" {
		t.Errorf("dry: %d %q %q", code, out, c.uploaded)
	}
	// upload for real
	c = mk()
	code, out, _ = run(t, c, nil, false, "package", "upload", "tool", "1.2", file, "--sha256", helloSum)
	if code != ExitOK || c.uploaded != "tool 1.2 tool-1.2.tgz" || c.data != "hello\n" || !c.linked || !strings.HasPrefix(out, "ok\t") {
		t.Errorf("upload: %d %q %+v", code, out, c)
	}
	// refusals before any request
	for name, a := range map[string][]string{
		"sum":      {"package", "upload", "tool", "1.2", file, "--sha256", strings.Repeat("0", 64)},
		"exists":   {"package", "upload", "tool", "1.1", file},
		"badsum":   {"package", "upload", "tool", "1.2", file, "--sha256", "XYZ"},
		"name":     {"package", "upload", "Tool", "1.2", file},
		"version":  {"package", "upload", "tool", "1..2", file},
		"missing":  {"package", "upload", "tool", "1.2", filepath.Join(dir, "none")},
		"symlink":  {"package", "upload", "tool", "1.2", link},
		"dir":      {"package", "upload", "tool", "1.2", dir},
		"empty":    {"package", "upload", "tool", "1.2", empty},
		"filename": {"package", "upload", "tool", "1.2", bad},
		"maxmib":   {"package", "upload", "tool", "1.2", file, "--max-mib", "0"},
		"stale":    {"package", "list", "--stale-days", "x"},
		"flag":     {"package", "list", "--dry-run"},
		"flag2":    {"package", "list", "--allow-newest"},
		"args":     {"package", "upload", "tool"},
		"listname": {"package", "list", "A"},
		"delargs":  {"package", "delete", "tool"},
	} {
		c := mk()
		code, _, _ := run(t, c, nil, false, a...)
		if code == ExitOK || c.uploaded != "" || c.deleted != "" {
			t.Errorf("%s: %d", name, code)
		}
	}
	if code, _, _ := run(t, mk(), nil, false, "package", "upload", "tool", "1.2", file, "--max-mib", "1", "--sha256", strings.Repeat("0", 64)); code != ExitFail {
		t.Errorf("sum code: %d", code)
	}
	big := filepath.Join(dir, "big.bin")
	os.WriteFile(big, make([]byte, 2*1024*1024), 0o644)
	if code, _, _ := run(t, mk(), nil, false, "package", "upload", "tool", "2.0", big, "--max-mib", "1"); code != ExitUsage {
		t.Errorf("big: %d", code)
	}
	// failures
	c = mk()
	c.uploadErr = forge.ErrNotFound
	if code, _, _ := run(t, c, nil, false, "package", "upload", "tool", "1.2", file); code != ExitFail {
		t.Errorf("upload err: %d", code)
	}
	c = mk()
	c.linkErr = forge.ErrNotFound
	if code, out, _ := run(t, c, nil, false, "package", "upload", "tool", "1.2", file); code != ExitFail || !strings.HasPrefix(out, "ok\t") {
		t.Errorf("link err: %d %q", code, out)
	}
	c = mk()
	c.err = forge.ErrNotFound
	if code, _, _ := run(t, c, nil, false, "package", "list"); code != ExitFail {
		t.Errorf("list err: %d", code)
	}

	// delete
	c = mk()
	if code, out, _ := run(t, c, nil, false, "package", "delete", "tool", "1.0"); code != ExitOK || out != "ok\n" || c.deleted != "tool 1.0" {
		t.Errorf("delete: %d %q %q", code, out, c.deleted)
	}
	c = mk()
	if code, _, errb := run(t, c, nil, false, "package", "delete", "tool", "1.1"); code != ExitFail || c.deleted != "" || !strings.Contains(errb, "--allow-newest") {
		t.Errorf("newest: %d %q", code, errb)
	}
	if code, _, _ := run(t, c, nil, false, "package", "delete", "tool", "1.1", "--allow-newest"); code != ExitOK || c.deleted != "tool 1.1" {
		t.Errorf("allow: %d", code)
	}
	if code, _, _ := run(t, mk(), nil, false, "package", "delete", "tool", "9.9"); code != ExitFail {
		t.Errorf("missing version: %d", code)
	}
	c = mk()
	c.pkgs = append(c.pkgs, forge.Package{Name: "tool", Version: "bad", Created: "oops"})
	c.err = nil
	if code, _, _ := run(t, c, nil, false, "package", "delete", "tool", "0.9"); code != ExitOK {
		t.Errorf("odd time: %d", code)
	}
	// no driver support
	if code, _, _ := run(t, fake{}, nil, false, "package", "list"); code != ExitUnavailable {
		t.Errorf("no driver: %d", code)
	}
}
