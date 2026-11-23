package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// Forgejo has no board API. These calls use the web routes, as forgejo-board.sh does. A read of
// a public repo needs no session. A write needs one.

var (
	colRe    = regexp.MustCompile(`class="project-column"[^>]*data-id="(\d+)"`)
	colName  = regexp.MustCompile(`project-column-title-label">([^<]*)<`)
	cardRe   = regexp.MustCompile(`class="issue-card[^"]*"[^>]*data-issue="(\d+)"`)
	cardLink = regexp.MustCompile(`class="issue-card-title[^"]*"[^>]*href="[^"]*/(?:issues|pulls)/(\d+)">([^<]*)<`)
)

// web calls a web route. It sends the session cookie when there is one. It sends no Origin
// header, so the cross-origin check passes without a token.
func (c *Client) web(ctx context.Context, method, path, ctype string, body []byte) (int, []byte, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+"/"+c.cfg.Repo+"/"+path, rd)
	if err != nil {
		return 0, nil, err
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	if c.cfg.Cookie != "" {
		req.AddCookie(&http.Cookie{Name: "i_like_gitea", Value: c.cfg.Cookie})
	}
	hc := *c.cfg.HTTP
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	return resp.StatusCode, out, err
}

func text(s string) string {
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// BoardList lists the boards, page by page. A page past the last repeats the last page, so it
// stops when a page holds nothing new.
func (c *Client) BoardList(ctx context.Context, closed bool) ([]forge.Board, error) {
	state := "open"
	if closed {
		state = "closed"
	}
	re := regexp.MustCompile(`<a class="muted tw-break-anywhere" href="/` + regexp.QuoteMeta(c.cfg.Repo) + `/projects/(\d+)">([^<]*)</a>`)
	var out []forge.Board
	seen := map[string]bool{}
	for page := 1; page <= 10; page++ {
		code, body, err := c.web(ctx, http.MethodGet, fmt.Sprintf("projects?state=%s&page=%d", state, page), "", nil)
		if err != nil {
			return nil, err
		}
		if code != http.StatusOK {
			return nil, &forge.Error{Status: code, Path: "projects", Body: string(body)}
		}
		fresh := 0
		for _, m := range re.FindAllStringSubmatch(string(body), -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			fresh++
			out = append(out, forge.Board{ID: m[1], Title: text(m[2]), URL: c.cfg.BaseURL + "/" + c.cfg.Repo + "/projects/" + m[1]})
		}
		if fresh == 0 {
			break
		}
	}
	return out, nil
}

func (c *Client) boardPage(ctx context.Context, board string) (string, []forge.Card, error) {
	code, body, err := c.web(ctx, http.MethodGet, "projects/"+url.PathEscape(board), "", nil)
	if err != nil {
		return "", nil, err
	}
	switch code {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", nil, fmt.Errorf("board %s: %w", board, forge.ErrNotFound)
	default:
		return "", nil, &forge.Error{Status: code, Path: "projects/" + board, Body: string(body)}
	}
	var cards []forge.Card
	var col, ctitle, iid string
	for _, line := range strings.Split(string(body), "\n") {
		if m := colRe.FindStringSubmatch(line); m != nil {
			col, ctitle, iid = m[1], "", ""
			continue
		}
		if col != "" && ctitle == "" {
			if m := colName.FindStringSubmatch(line); m != nil {
				ctitle = text(m[1])
				continue
			}
		}
		if m := cardRe.FindStringSubmatch(line); m != nil {
			iid = m[1]
			continue
		}
		if iid != "" {
			if m := cardLink.FindStringSubmatch(line); m != nil {
				n, _ := strconv.Atoi(m[1])
				cards = append(cards, forge.Card{ColumnID: col, Column: ctitle, ItemID: iid, Number: n, Title: text(m[2])})
				iid = ""
			}
		}
	}
	return string(body), cards, nil
}

// BoardView lists the cards of a board. A column with no cards gives no row.
func (c *Client) BoardView(ctx context.Context, board string) ([]forge.Card, error) {
	_, cards, err := c.boardPage(ctx, board)
	return cards, err
}

func (c *Client) issueID(ctx context.Context, n int) (int64, error) {
	path := fmt.Sprintf("repos/%s/issues/%d", c.cfg.Repo, n)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/"+path, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)
	body, err := c.send(req, path)
	if err != nil {
		return 0, err
	}
	var v struct {
		ID int64 `json:"id"`
	}
	if json.Unmarshal(body, &v) != nil || v.ID == 0 {
		return 0, fmt.Errorf("issue %d: %w", n, forge.ErrNotFound)
	}
	return v.ID, nil
}

// checkWrite reads the answer of a board write.
func checkWrite(code int, body []byte, path string) error {
	switch code {
	case http.StatusOK:
		var v struct {
			OK bool `json:"ok"`
		}
		if json.Unmarshal(body, &v) != nil || !v.OK {
			return fmt.Errorf("the server did not answer ok: %.200s", body)
		}
		return nil
	case http.StatusSeeOther, http.StatusUnauthorized, http.StatusForbidden:
		return forge.ErrNoSession
	}
	return &forge.Error{Status: code, Path: path, Body: string(body)}
}

// BoardAdd puts issues on a board.
func (c *Client) BoardAdd(ctx context.Context, board string, numbers []int) error {
	if c.cfg.Cookie == "" {
		return forge.ErrNoSession
	}
	var ids []string
	for _, n := range numbers {
		id, err := c.issueID(ctx, n)
		if err != nil {
			return err
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	form := url.Values{"id": {board}, "issue_ids": {strings.Join(ids, ",")}}
	code, body, err := c.web(ctx, http.MethodPost, "issues/projects", "application/x-www-form-urlencoded", []byte(form.Encode()))
	if err != nil {
		return err
	}
	return checkWrite(code, body, "issues/projects")
}

// BoardMove moves cards to a column. The body lists the whole column in order, so the cards that
// stay come first and the moved cards come last.
func (c *Client) BoardMove(ctx context.Context, board, column string, numbers []int) error {
	if c.cfg.Cookie == "" {
		return forge.ErrNoSession
	}
	page, cards, err := c.boardPage(ctx, board)
	if err != nil {
		return err
	}
	col := ""
	for _, k := range cards {
		if k.ColumnID == column {
			col = column
		}
	}
	if col == "" {
		for _, k := range cards {
			if strings.EqualFold(k.Column, column) {
				col = k.ColumnID
				break
			}
		}
	}
	if col == "" { // a column with no cards has no row
		var cur string
		for _, line := range strings.Split(page, "\n") {
			if m := colRe.FindStringSubmatch(line); m != nil {
				cur = m[1]
			}
			if m := colName.FindStringSubmatch(line); m != nil && (strings.EqualFold(text(m[1]), column) || cur == column) {
				col = cur
				break
			}
		}
	}
	if col == "" {
		return fmt.Errorf("column %s on board %s: %w", column, board, forge.ErrNotFound)
	}
	moving := map[string]bool{}
	var moved []string
	for _, n := range numbers {
		id, err := c.issueID(ctx, n)
		if err != nil {
			return err
		}
		s := strconv.FormatInt(id, 10)
		moving[s] = true
		moved = append(moved, s)
	}
	var order []string
	for _, k := range cards {
		if k.ColumnID == col && !moving[k.ItemID] {
			order = append(order, k.ItemID)
		}
	}
	order = append(order, moved...)
	type item struct {
		IssueID int64 `json:"issueID"`
		Sorting int   `json:"sorting"`
	}
	var payload struct {
		Issues []item `json:"issues"`
	}
	for i, s := range order {
		id, _ := strconv.ParseInt(s, 10, 64)
		payload.Issues = append(payload.Issues, item{IssueID: id, Sorting: i})
	}
	b, _ := json.Marshal(payload)
	path := fmt.Sprintf("projects/%s/%s/move", board, col)
	code, body, err := c.web(ctx, http.MethodPost, path, "application/json", b)
	if err != nil {
		return err
	}
	return checkWrite(code, body, path)
}
