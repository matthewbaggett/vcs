package cli

import lexicli "git.grey.ooo/grey.ooo/lexi/cli"

// lexiMain runs the lexi command line. `vcs lexi` and `vcs alex` pass every argument through
// and return the exit code unchanged, so they match the lexi binary. Both skip the flag
// parser, the verb hints, and the prose checkers, because none of them may change the output.
func lexiMain(args []string, env Env) int {
	return lexicli.Run(args, env.Stdin, env.Stdout, env.Stderr)
}
