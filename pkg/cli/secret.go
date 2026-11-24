package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// secretName is the form that Forgejo accepts for a secret name.
var secretName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// secretSetCheck reads the value of "secret set" from a file, from standard input, or from an
// environment variable. The value never comes from an argument, so it stays out of the process
// list and the shell history. The text goes in o.flags["value"]. It returns -1 to go on.
func secretSetCheck(env Env, v Verb, o *opts) int {
	vf, fe := o.flags["value-file"], o.flags["from-env"]
	if len(o.pos) != 1 || !secretName.MatchString(o.pos[0]) || (vf == "") == (fe == "") {
		return usage(env, v)
	}
	var val string
	if fe != "" {
		val = env.Getenv(fe)
	} else {
		var raw []byte
		var err error
		if vf == "-" {
			if env.Stdin == nil {
				return usage(env, v)
			}
			raw, err = io.ReadAll(env.Stdin)
		} else {
			raw, err = os.ReadFile(vf)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "vcs: cannot read a secret value from %s\n", vf)
			return ExitUsage
		}
		val = string(raw)
	}
	val = strings.TrimRight(val, "\r\n")
	if val == "" {
		fmt.Fprintln(env.Stderr, "vcs: the secret value is empty. Nothing was sent")
		return ExitUsage
	}
	o.flags["value"] = val
	return -1
}
