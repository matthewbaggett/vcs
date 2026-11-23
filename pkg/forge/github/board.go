package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

// GitHub Projects v2 has GraphQL only. A board id is the node id of the project. A board that
// the user names by number is resolved to the node id first. The column is the Status field.

const statusField = "Status"

func (c *Client) graphqlURL() string {
	if c.cfg.GraphQLURL != "" {
		return c.cfg.GraphQLURL
	}
	if strings.HasSuffix(c.cfg.BaseURL, "/api/v3") { // GitHub Enterprise
		return strings.TrimSuffix(c.cfg.BaseURL, "/v3") + "/graphql"
	}
	return c.cfg.BaseURL + "/graphql"
}

func (c *Client) repoVars(extra map[string]any) map[string]any {
	owner, name, _ := strings.Cut(c.cfg.Repo, "/")
	v := map[string]any{"owner": owner, "name": name}
	for k, x := range extra {
		v[k] = x
	}
	return v
}

// gql runs one query. out may be nil.
func (c *Client) gql(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := c.send(ctx, http.MethodPost, c.graphqlURL(), map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("the server did not return a GraphQL answer")
	}
	if len(env.Errors) > 0 {
		if env.Errors[0].Type == "NOT_FOUND" {
			return fmt.Errorf("%s: %w", env.Errors[0].Message, forge.ErrNotFound)
		}
		return fmt.Errorf("GraphQL: %s", env.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

type projectJSON struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Closed bool   `json:"closed"`
}

// BoardList lists the projects of the repository.
func (c *Client) BoardList(ctx context.Context, closed bool) ([]forge.Board, error) {
	var out []forge.Board
	after := ""
	for page := 0; page < 20; page++ {
		var q struct {
			Repository struct {
				Projects struct {
					Nodes    []projectJSON `json:"nodes"`
					PageInfo struct {
						HasNext bool   `json:"hasNextPage"`
						End     string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"projectsV2"`
			} `json:"repository"`
		}
		vars := c.repoVars(nil)
		if after != "" {
			vars["after"] = after
		}
		if err := c.gql(ctx, `query($owner:String!,$name:String!,$after:String){repository(owner:$owner,name:$name){projectsV2(first:50,after:$after){nodes{id title url closed} pageInfo{hasNextPage endCursor}}}}`, vars, &q); err != nil {
			return nil, err
		}
		for _, p := range q.Repository.Projects.Nodes {
			if p.Closed == closed {
				out = append(out, forge.Board{ID: p.ID, Title: p.Title, URL: p.URL})
			}
		}
		if !q.Repository.Projects.PageInfo.HasNext {
			break
		}
		after = q.Repository.Projects.PageInfo.End
	}
	return out, nil
}

// projectID turns a board number into the node id. A node id passes through.
func (c *Client) projectID(ctx context.Context, board string) (string, error) {
	if _, err := strconv.Atoi(board); err != nil {
		return board, nil
	}
	var q struct {
		Repository struct {
			Project *struct {
				ID string `json:"id"`
			} `json:"projectV2"`
		} `json:"repository"`
	}
	n, _ := strconv.Atoi(board)
	if err := c.gql(ctx, `query($owner:String!,$name:String!,$n:Int!){repository(owner:$owner,name:$name){projectV2(number:$n){id}}}`, c.repoVars(map[string]any{"n": n}), &q); err != nil {
		return "", err
	}
	if q.Repository.Project == nil || q.Repository.Project.ID == "" {
		return "", fmt.Errorf("board %s: %w", board, forge.ErrNotFound)
	}
	return q.Repository.Project.ID, nil
}

// BoardView lists the cards of one board. A card with no Status has an empty column. A draft
// card has no number, so it gives no row.
func (c *Client) BoardView(ctx context.Context, board string) ([]forge.Card, error) {
	id, err := c.projectID(ctx, board)
	if err != nil {
		return nil, err
	}
	cards, err := c.items(ctx, id)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func (c *Client) items(ctx context.Context, project string) ([]forge.Card, error) {
	var out []forge.Card
	after := ""
	for page := 0; page < 50; page++ {
		var q struct {
			Node *struct {
				Items struct {
					Nodes []struct {
						ID      string `json:"id"`
						Content struct {
							Number int    `json:"number"`
							Title  string `json:"title"`
						} `json:"content"`
						Status struct {
							Name     string `json:"name"`
							OptionID string `json:"optionId"`
						} `json:"fieldValueByName"`
					} `json:"nodes"`
					PageInfo struct {
						HasNext bool   `json:"hasNextPage"`
						End     string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"items"`
			} `json:"node"`
		}
		vars := map[string]any{"id": project, "field": statusField}
		if after != "" {
			vars["after"] = after
		}
		const query = `query($id:ID!,$field:String!,$after:String){node(id:$id){... on ProjectV2{items(first:100,after:$after){nodes{id content{... on Issue{number title} ... on PullRequest{number title}} fieldValueByName(name:$field){... on ProjectV2ItemFieldSingleSelectValue{name optionId}}} pageInfo{hasNextPage endCursor}}}}}`
		if err := c.gql(ctx, query, vars, &q); err != nil {
			return nil, err
		}
		if q.Node == nil {
			return nil, fmt.Errorf("board %s: %w", project, forge.ErrNotFound)
		}
		for _, n := range q.Node.Items.Nodes {
			if n.Content.Number == 0 {
				continue
			}
			out = append(out, forge.Card{ColumnID: n.Status.OptionID, Column: n.Status.Name, ItemID: n.ID, Number: n.Content.Number, Title: n.Content.Title})
		}
		if !q.Node.Items.PageInfo.HasNext {
			break
		}
		after = q.Node.Items.PageInfo.End
	}
	return out, nil
}

// contentID reads the node id of an issue or a pull request.
func (c *Client) contentID(ctx context.Context, n int) (string, error) {
	var q struct {
		Repository struct {
			Item *struct {
				ID string `json:"id"`
			} `json:"issueOrPullRequest"`
		} `json:"repository"`
	}
	if err := c.gql(ctx, `query($owner:String!,$name:String!,$n:Int!){repository(owner:$owner,name:$name){issueOrPullRequest(number:$n){... on Issue{id} ... on PullRequest{id}}}}`, c.repoVars(map[string]any{"n": n}), &q); err != nil {
		return "", err
	}
	if q.Repository.Item == nil || q.Repository.Item.ID == "" {
		return "", fmt.Errorf("#%d: %w", n, forge.ErrNotFound)
	}
	return q.Repository.Item.ID, nil
}

func (c *Client) addItem(ctx context.Context, project string, n int) (string, error) {
	content, err := c.contentID(ctx, n)
	if err != nil {
		return "", err
	}
	var q struct {
		Add struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	if err := c.gql(ctx, `mutation($p:ID!,$c:ID!){addProjectV2ItemById(input:{projectId:$p,contentId:$c}){item{id}}}`, map[string]any{"p": project, "c": content}, &q); err != nil {
		return "", err
	}
	return q.Add.Item.ID, nil
}

// BoardAdd puts issues or pull requests on a board. A card that is on the board stays as it is.
func (c *Client) BoardAdd(ctx context.Context, board string, numbers []int) error {
	id, err := c.projectID(ctx, board)
	if err != nil {
		return err
	}
	for _, n := range numbers {
		if _, err := c.addItem(ctx, id, n); err != nil {
			return err
		}
	}
	return nil
}

// BoardMove sets the Status of cards. The column is an option id or an option title. A card that
// is not on the board is added first.
func (c *Client) BoardMove(ctx context.Context, board, column string, numbers []int) error {
	id, err := c.projectID(ctx, board)
	if err != nil {
		return err
	}
	var q struct {
		Node *struct {
			Field *struct {
				ID      string `json:"id"`
				Options []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"options"`
			} `json:"field"`
		} `json:"node"`
	}
	if err := c.gql(ctx, `query($id:ID!,$field:String!){node(id:$id){... on ProjectV2{field(name:$field){... on ProjectV2SingleSelectField{id options{id name}}}}}}`, map[string]any{"id": id, "field": statusField}, &q); err != nil {
		return err
	}
	if q.Node == nil || q.Node.Field == nil || q.Node.Field.ID == "" {
		return fmt.Errorf("field %s on board %s: %w", statusField, board, forge.ErrNotFound)
	}
	option := ""
	for _, o := range q.Node.Field.Options {
		if o.ID == column {
			option = o.ID
		}
	}
	if option == "" {
		for _, o := range q.Node.Field.Options {
			if strings.EqualFold(o.Name, column) {
				option = o.ID
				break
			}
		}
	}
	if option == "" {
		return fmt.Errorf("column %s on board %s: %w", column, board, forge.ErrNotFound)
	}
	cards, err := c.items(ctx, id)
	if err != nil {
		return err
	}
	onBoard := map[int]string{}
	for _, k := range cards {
		onBoard[k.Number] = k.ItemID
	}
	for _, n := range numbers {
		item, ok := onBoard[n]
		if !ok {
			if item, err = c.addItem(ctx, id, n); err != nil {
				return err
			}
		}
		err := c.gql(ctx, `mutation($p:ID!,$i:ID!,$f:ID!,$o:String!){updateProjectV2ItemFieldValue(input:{projectId:$p,itemId:$i,fieldId:$f,value:{singleSelectOptionId:$o}}){projectV2Item{id}}}`,
			map[string]any{"p": id, "i": item, "f": q.Node.Field.ID, "o": option}, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
