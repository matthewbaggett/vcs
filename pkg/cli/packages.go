package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// The scope rules for the three names that reach the address of a request.
var (
	pkgNameRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	pkgVersionRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)
	pkgFileRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)
	sha256Re     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// packageCheck checks the arguments of the three package verbs. It reads and hashes the file of an
// upload, and keeps the facts in the hidden flags _file, _sha, and _size. A user can't set a flag
// that starts with an underscore. It returns -1 to go on.
func packageCheck(env Env, v Verb, o *opts) int {
	if s := o.flags["stale-days"]; s != "" {
		if n, err := strconv.Atoi(s); err != nil || n < 0 {
			fmt.Fprintln(env.Stderr, "vcs: --stale-days takes a whole number of days")
			return ExitUsage
		}
	}
	if (o.flags["max-mib"] != "" || o.flags["sha256"] != "" || o.flags["dry-run"] != "") && v.Name != "upload" {
		return usage(env, v)
	}
	if o.flags["allow-newest"] != "" && v.Name != "delete" {
		return usage(env, v)
	}
	switch v.Name {
	case "list":
		if len(o.pos) > 1 || (len(o.pos) == 1 && !pkgNameRe.MatchString(o.pos[0])) {
			return usage(env, v)
		}
	case "delete":
		if len(o.pos) != 2 || !pkgNameRe.MatchString(o.pos[0]) || !pkgVersionRe.MatchString(o.pos[1]) || strings.Contains(o.pos[1], "..") {
			return usage(env, v)
		}
	case "upload":
		if len(o.pos) != 3 || !pkgNameRe.MatchString(o.pos[0]) || !pkgVersionRe.MatchString(o.pos[1]) || strings.Contains(o.pos[1], "..") {
			return usage(env, v)
		}
		maxMiB := 512
		if s := o.flags["max-mib"]; s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 {
				fmt.Fprintln(env.Stderr, "vcs: --max-mib takes a whole number above zero")
				return ExitUsage
			}
			maxMiB = n
		}
		if s := o.flags["sha256"]; s != "" && !sha256Re.MatchString(s) {
			fmt.Fprintln(env.Stderr, "vcs: --sha256 takes 64 lower-case hex characters")
			return ExitUsage
		}
		file := o.pos[2]
		fi, err := os.Lstat(file)
		switch {
		case err != nil:
			fmt.Fprintf(env.Stderr, "vcs: no such file: %s\n", file)
			return ExitUsage
		case fi.Mode()&os.ModeSymlink != 0:
			fmt.Fprintf(env.Stderr, "vcs: the file is a link. Give the file itself: %s\n", file)
			return ExitUsage
		case !fi.Mode().IsRegular():
			fmt.Fprintf(env.Stderr, "vcs: not a plain file: %s\n", file)
			return ExitUsage
		}
		base := filepath.Base(file)
		if !pkgFileRe.MatchString(base) || strings.Contains(base, "..") {
			fmt.Fprintf(env.Stderr, "vcs: the file name %s is not allowed. Use letters, digits, '.', '_', '+' and '-', and no '..'. Rename a copy\n", base)
			return ExitUsage
		}
		if fi.Size() == 0 {
			fmt.Fprintf(env.Stderr, "vcs: the file is empty: %s\n", file)
			return ExitUsage
		}
		if fi.Size() > int64(maxMiB)*1024*1024 {
			fmt.Fprintf(env.Stderr, "vcs: the file is %d MiB, over the limit of %d MiB. Raise it with --max-mib on purpose\n", fi.Size()/1024/1024, maxMiB)
			return ExitUsage
		}
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: cannot read %s\n", file)
			return ExitUsage
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: cannot read %s\n", file)
			return ExitUsage
		}
		sum := hex.EncodeToString(h.Sum(nil))
		if want := o.flags["sha256"]; want != "" && want != sum {
			fmt.Fprintf(env.Stderr, "vcs: the checksum of the file is %s, not %s. Nothing was sent\n", sum, want)
			return ExitFail
		}
		o.flags["_file"], o.flags["_sha"], o.flags["_size"] = base, sum, strconv.FormatInt(fi.Size(), 10)
	}
	return -1
}

// packageRows turns the versions of the registry into list rows. Each name keeps its versions
// newest first. The newest is "newest". Any other version at least stale days old is "old". The
// rest are "recent".
func packageRows(ps []forge.Package, now time.Time, stale int) [][]string {
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Name != ps[j].Name {
			return ps[i].Name < ps[j].Name
		}
		return ps[i].Created > ps[j].Created
	})
	var rows [][]string
	prev := ""
	for i, p := range ps {
		age := 0
		if t, err := time.Parse(time.RFC3339, p.Created); err == nil {
			age = int(now.Sub(t).Hours() / 24)
		}
		status := "recent"
		switch {
		case i == 0 || p.Name != prev:
			status = "newest"
		case age >= stale:
			status = "old"
		}
		prev = p.Name
		rows = append(rows, []string{p.Name, p.Version, p.Created, strconv.Itoa(age), status})
	}
	return rows
}

// packageNotice says on stderr how many old versions of a name the rows hold.
func packageNotice(env Env, name string, rows [][]string, stale int) {
	old := 0
	for _, r := range rows {
		if r[4] == "old" {
			old++
		}
	}
	if old > 0 {
		fmt.Fprintf(env.Stderr, "vcs: %d older version(s) of %s are at least %d days old and may need cleanup. Remove one on purpose with: vcs package delete %s <version>\n", old, name, stale, name)
	}
}

func packageRun(ctx context.Context, c forge.Client, env Env, v Verb, o opts) int {
	pm, ok := c.(forge.PackageManager)
	if !ok {
		fmt.Fprintln(env.Stderr, "vcs: this host has no package verbs yet")
		return ExitUnavailable
	}
	stale := 30
	if s := o.flags["stale-days"]; s != "" {
		stale, _ = strconv.Atoi(s)
	}
	now := time.Now()
	if env.Now != nil {
		now = env.Now()
	}
	name := ""
	if len(o.pos) > 0 {
		name = o.pos[0]
	}
	ps, err := pm.PackageList(ctx, name)
	if err != nil {
		return fail(env, err)
	}
	rows := packageRows(ps, now, stale)
	switch v.Name {
	case "list":
		for _, r := range rows {
			row(env, r...)
		}
		if name != "" {
			packageNotice(env, name, rows, stale)
		}
	case "upload":
		version := o.pos[1]
		for _, r := range rows {
			if r[1] == version {
				fmt.Fprintf(env.Stderr, "vcs: %s %s already exists. An upload never replaces. Pick a new version, or delete this one on purpose\n", name, version)
				return ExitFail
			}
		}
		url := pm.PackageURL(name, version, o.flags["_file"])
		if o.flags["dry-run"] == "true" {
			row(env, "dry-run", url, o.flags["_sha"])
			for _, r := range rows {
				row(env, r...)
			}
			packageNotice(env, name, rows, stale)
			return ExitOK
		}
		f, err := os.Open(o.pos[2])
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: cannot read %s\n", o.pos[2])
			return ExitFail
		}
		defer f.Close()
		size, _ := strconv.ParseInt(o.flags["_size"], 10, 64)
		if err := pm.PackageUpload(ctx, name, version, o.flags["_file"], f, size); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: the upload of %s to %s %s failed\n", o.flags["_file"], name, version)
			return fail(env, err)
		}
		row(env, "ok", url, o.flags["_sha"])
		if err := pm.PackageLink(ctx, name); err != nil {
			fmt.Fprintf(env.Stderr, "vcs: the file is uploaded, but the link to the repository failed. Link it by hand, or delete %s %s\n", name, version)
			return fail(env, err)
		}
		ps, err := pm.PackageList(ctx, name)
		if err != nil {
			return fail(env, err)
		}
		rows = packageRows(ps, now, stale)
		for _, r := range rows {
			row(env, r...)
		}
		packageNotice(env, name, rows, stale)
	case "delete":
		version := o.pos[1]
		status := ""
		for _, r := range rows {
			if r[1] == version {
				status = r[4]
			}
		}
		if status == "" {
			fmt.Fprintf(env.Stderr, "vcs: %s %s is not in the registry of the owner\n", name, version)
			return ExitFail
		}
		if status == "newest" && o.flags["allow-newest"] != "true" {
			fmt.Fprintf(env.Stderr, "vcs: %s %s is the newest version. Pass --allow-newest to delete it on purpose\n", name, version)
			return ExitFail
		}
		if err := pm.PackageDelete(ctx, name, version); err != nil {
			return fail(env, err)
		}
		fmt.Fprintln(env.Stdout, "ok")
	}
	return ExitOK
}
