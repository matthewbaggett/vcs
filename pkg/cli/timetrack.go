package cli

import (
	"fmt"
	"regexp"
	"strconv"
)

var durationRe = regexp.MustCompile(`^(?:([0-9]+)h)?(?:([0-9]+)m)?(?:([0-9]+)s)?$`)

// parseDuration reads hours, minutes, and seconds, each optional, in that order. For example
// 1h25m, 90m, 45s, or 2h5m30s. It returns false for an empty text, a bad text, or zero.
func parseDuration(s string) (int64, bool) {
	m := durationRe.FindStringSubmatch(s)
	if s == "" || m == nil {
		return 0, false
	}
	var total int64
	for i, unit := range []int64{3600, 60, 1} {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.ParseInt(m[i+1], 10, 64)
		if err != nil {
			return 0, false
		}
		total += n * unit
	}
	return total, total > 0
}

// hms prints seconds as 1h25m0s. It always has all three parts.
func hms(sec int64) string {
	return fmt.Sprintf("%dh%dm%ds", sec/3600, sec%3600/60, sec%60)
}

// timeCheck checks the arguments of the time and stopwatch verbs. It returns an exit code, or -1 to go on.
func timeCheck(env Env, v Verb, o opts) int {
	switch v.Noun + " " + v.Name {
	case "time add":
		if len(o.pos) != 2 || !isNum(o.pos[0]) {
			return usage(env, v)
		}
		if _, ok := parseDuration(o.pos[1]); !ok {
			return usage(env, v)
		}
	case "time list", "time total":
		if len(o.pos) > 1 || (len(o.pos) == 1 && !isNum(o.pos[0])) {
			return usage(env, v)
		}
	case "stopwatch list":
		if len(o.pos) != 0 {
			return usage(env, v)
		}
	}
	return -1
}
