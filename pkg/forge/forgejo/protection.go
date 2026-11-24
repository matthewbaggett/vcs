package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// ProtectionList reads every branch protection rule of the repository. The call needs a token,
// even on a public repository. A 401 or 403 gives ErrNoSession.
func (c *Client) ProtectionList(ctx context.Context) ([]forge.Protection, error) {
	var out []forge.Protection
	for page := 1; page <= 100; page++ {
		body, err := c.get(ctx, fmt.Sprintf("repos/%s/branch_protections?limit=50&page=%d", c.cfg.Repo, page))
		if err != nil {
			var fe *forge.Error
			if errors.As(err, &fe) && (fe.Status == 401 || fe.Status == 403) {
				return nil, fmt.Errorf("%w. The server refused the token: %s", forge.ErrNoSession, fe.Body)
			}
			return nil, err
		}
		var rows []struct {
			RuleName          string   `json:"rule_name"`
			BranchName        string   `json:"branch_name"`
			EnableStatusCheck bool     `json:"enable_status_check"`
			Contexts          []string `json:"status_check_contexts"`
			Approvals         int      `json:"required_approvals"`
			EnablePush        bool     `json:"enable_push"`
			PushWhitelist     bool     `json:"enable_push_whitelist"`
			ApplyToAdmins     bool     `json:"apply_to_admins"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("the server did not return a list of rules")
		}
		for _, r := range rows {
			p := forge.Protection{Rule: r.RuleName, Approvals: r.Approvals, Admins: r.ApplyToAdmins}
			if p.Rule == "" {
				p.Rule = r.BranchName
			}
			if r.EnableStatusCheck {
				p.Contexts = r.Contexts
			}
			switch {
			case !r.EnablePush:
				p.Push = "none"
			case r.PushWhitelist:
				p.Push = "whitelist"
			default:
				p.Push = "all"
			}
			out = append(out, p)
		}
		if len(rows) < 50 {
			break
		}
	}
	return out, nil
}
