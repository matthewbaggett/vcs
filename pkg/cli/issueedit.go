package cli

import (
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// splitList splits a comma-separated flag value and drops the empty items.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// issueEditEmpty reports whether the flags of an issue edit change nothing.
func issueEditEmpty(o opts) bool {
	for _, k := range []string{"title", "body", "body-file", "state", "milestone", "add-label", "rm-label"} {
		if o.flags[k] != "" {
			return false
		}
	}
	return true
}

// issueEditInput turns the flags of an issue edit into the driver input.
func issueEditInput(o opts) forge.IssueEdit {
	var in forge.IssueEdit
	for k, dst := range map[string]**string{"title": &in.Title, "body": &in.Body, "state": &in.State, "milestone": &in.Milestone} {
		if v := o.flags[k]; v != "" {
			s := v
			*dst = &s
		}
	}
	in.AddLabels = splitList(o.flags["add-label"])
	in.RmLabels = splitList(o.flags["rm-label"])
	return in
}
