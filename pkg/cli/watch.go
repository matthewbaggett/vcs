package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// settled reports whether a pull request is done, and whether the end is good.
// A good end is a merged pull request, or an open one with green checks and no conflict.
// A failed check, a closed pull request, or a conflict is a bad end that needs a fix.
func settled(s forge.PRStatus) (done, good bool, why string) {
	switch {
	case s.State == "merged":
		return true, true, "merged"
	case s.State == "closed":
		return true, false, "the pull request is closed"
	case s.CI == "failure":
		return true, false, "a check failed"
	case len(s.ConflictedFiles) > 0:
		return true, false, "the pull request has a conflict"
	case s.CI == "success" && (s.Mergeable || s.Draft):
		return true, true, "checks are green and the merge state is clean"
	}
	return false, false, ""
}

// watch polls the pull request until it settles or the timeout ends.
func watch(ctx context.Context, c forge.Client, env Env, n int, o opts) int {
	interval, _ := intFlag(o, "interval", 30)
	timeout, _ := intFlag(o, "timeout", 3600)
	sleep := env.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) {
			select {
			case <-ctx.Done():
			case <-time.After(d):
			}
		}
	}
	waited := 0
	last := ""
	for {
		s, err := c.PRStatus(ctx, n)
		if err != nil {
			return fail(env, err)
		}
		line := strings.Join([]string{s.State, strconv.FormatBool(s.Draft), strconv.FormatBool(s.Mergeable),
			strings.Join(s.ConflictedFiles, ","), s.CI, s.HeadSHA, s.URL}, "\t")
		if line != last {
			row(env, s.State, strconv.FormatBool(s.Draft), strconv.FormatBool(s.Mergeable),
				strings.Join(s.ConflictedFiles, ","), s.CI, s.HeadSHA, s.URL)
			last = line
		}
		if done, good, why := settled(s); done {
			fmt.Fprintf(env.Stderr, "vcs: pr watch: %s\n", why)
			if good {
				return ExitOK
			}
			return ExitFail
		}
		if waited+interval > timeout {
			fmt.Fprintf(env.Stderr, "vcs: pr watch: timed out after %d seconds\n", waited)
			return ExitFail
		}
		if ctx.Err() != nil {
			return ExitFail
		}
		sleep(ctx, time.Duration(interval)*time.Second)
		waited += interval
	}
}
