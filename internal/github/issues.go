package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
)

type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Author    Actor      `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	Labels    []Label    `json:"labels"`
	Comments  []Comment  `json:"comments"`
	Events    []Event    `json:"events"`
}

type Actor struct {
	Login string `json:"login"`
}

type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Comment struct {
	Author    Actor     `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	Type      string    `json:"type"`
	Actor     Actor     `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	Details   any       `json:"details,omitempty"`
}

type issueQuery struct {
	Repository struct {
		Issues struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []struct {
				Number    githubv4.Int
				Title     githubv4.String
				Body      githubv4.String
				State     githubv4.String
				CreatedAt githubv4.DateTime
				UpdatedAt githubv4.DateTime
				ClosedAt  *githubv4.DateTime
				Author    struct {
					Login githubv4.String
				}
				Labels struct {
					Nodes []struct {
						Name  githubv4.String
						Color githubv4.String
					}
				} `graphql:"labels(first: 50)"`
				Comments struct {
					Nodes []struct {
						Author struct {
							Login githubv4.String
						}
						Body      githubv4.String
						CreatedAt githubv4.DateTime
						UpdatedAt githubv4.DateTime
					}
				} `graphql:"comments(first: 50)"`
				TimelineItems struct {
					Nodes []issueTimelineItem
				} `graphql:"timelineItems(first: 50)"`
			}
		} `graphql:"issues(first: 20, orderBy: {field: UPDATED_AT, direction: DESC}, filterBy: {since: $since}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

type closedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
}

type reopenedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
}

type labeledEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
	Label     struct{ Name githubv4.String }
}

type unlabeledEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
	Label     struct{ Name githubv4.String }
}

type assignedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
	Assignee  struct {
		User struct{ Login githubv4.String } `graphql:"... on User"`
	}
}

type unassignedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
	Assignee  struct {
		User struct{ Login githubv4.String } `graphql:"... on User"`
	}
}

type crossReferencedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
	Source    struct {
		Issue struct{ Number githubv4.Int } `graphql:"... on Issue"`
		PR    struct{ Number githubv4.Int } `graphql:"... on PullRequest"`
	}
}

type issueTimelineItem struct {
	TypeName             string               `graphql:"__typename"`
	ClosedEvent          closedEvent          `graphql:"... on ClosedEvent"`
	ReopenedEvent        reopenedEvent        `graphql:"... on ReopenedEvent"`
	LabeledEvent         labeledEvent         `graphql:"... on LabeledEvent"`
	UnlabeledEvent       unlabeledEvent       `graphql:"... on UnlabeledEvent"`
	AssignedEvent        assignedEvent        `graphql:"... on AssignedEvent"`
	UnassignedEvent      unassignedEvent      `graphql:"... on UnassignedEvent"`
	CrossReferencedEvent crossReferencedEvent `graphql:"... on CrossReferencedEvent"`
}

func (c *Client) FetchIssues(ctx context.Context, owner, repo string, since *time.Time) ([]Issue, error) {
	var allIssues []Issue
	var cursor *githubv4.String

	var sinceDateTime *githubv4.DateTime
	if since != nil {
		dt := githubv4.DateTime{Time: *since}
		sinceDateTime = &dt
	}

	for {
		var q issueQuery
		vars := map[string]any{
			"owner":  githubv4.String(owner),
			"repo":   githubv4.String(repo),
			"since":  sinceDateTime,
			"cursor": cursor,
		}

		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, err
		}

		for _, node := range q.Repository.Issues.Nodes {
			issue := Issue{
				Number:    int(node.Number),
				Title:     string(node.Title),
				Body:      string(node.Body),
				State:     string(node.State),
				Author:    Actor{Login: string(node.Author.Login)},
				CreatedAt: node.CreatedAt.Time,
				UpdatedAt: node.UpdatedAt.Time,
			}

			if node.ClosedAt != nil {
				t := node.ClosedAt.Time
				issue.ClosedAt = &t
			}

			for _, l := range node.Labels.Nodes {
				issue.Labels = append(issue.Labels, Label{
					Name:  string(l.Name),
					Color: string(l.Color),
				})
			}

			for _, c := range node.Comments.Nodes {
				issue.Comments = append(issue.Comments, Comment{
					Author:    Actor{Login: string(c.Author.Login)},
					Body:      string(c.Body),
					CreatedAt: c.CreatedAt.Time,
					UpdatedAt: c.UpdatedAt.Time,
				})
			}

			for _, ti := range node.TimelineItems.Nodes {
				event := convertTimelineEvent(ti)
				if event != nil {
					issue.Events = append(issue.Events, *event)
				}
			}

			allIssues = append(allIssues, issue)
		}

		if !q.Repository.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = &q.Repository.Issues.PageInfo.EndCursor
	}

	return allIssues, nil
}

func convertTimelineEvent(ti issueTimelineItem) *Event {
	switch ti.TypeName {
	case "ClosedEvent":
		return &Event{
			Type:      "closed",
			Actor:     Actor{Login: string(ti.ClosedEvent.Actor.Login)},
			CreatedAt: ti.ClosedEvent.CreatedAt.Time,
		}
	case "ReopenedEvent":
		return &Event{
			Type:      "reopened",
			Actor:     Actor{Login: string(ti.ReopenedEvent.Actor.Login)},
			CreatedAt: ti.ReopenedEvent.CreatedAt.Time,
		}
	case "LabeledEvent":
		return &Event{
			Type:      "labeled",
			Actor:     Actor{Login: string(ti.LabeledEvent.Actor.Login)},
			CreatedAt: ti.LabeledEvent.CreatedAt.Time,
			Details:   map[string]string{"label": string(ti.LabeledEvent.Label.Name)},
		}
	case "UnlabeledEvent":
		return &Event{
			Type:      "unlabeled",
			Actor:     Actor{Login: string(ti.UnlabeledEvent.Actor.Login)},
			CreatedAt: ti.UnlabeledEvent.CreatedAt.Time,
			Details:   map[string]string{"label": string(ti.UnlabeledEvent.Label.Name)},
		}
	case "AssignedEvent":
		return &Event{
			Type:      "assigned",
			Actor:     Actor{Login: string(ti.AssignedEvent.Actor.Login)},
			CreatedAt: ti.AssignedEvent.CreatedAt.Time,
			Details:   map[string]string{"assignee": string(ti.AssignedEvent.Assignee.User.Login)},
		}
	case "UnassignedEvent":
		return &Event{
			Type:      "unassigned",
			Actor:     Actor{Login: string(ti.UnassignedEvent.Actor.Login)},
			CreatedAt: ti.UnassignedEvent.CreatedAt.Time,
			Details:   map[string]string{"assignee": string(ti.UnassignedEvent.Assignee.User.Login)},
		}
	case "CrossReferencedEvent":
		return &Event{
			Type:      "cross-referenced",
			Actor:     Actor{Login: string(ti.CrossReferencedEvent.Actor.Login)},
			CreatedAt: ti.CrossReferencedEvent.CreatedAt.Time,
		}
	default:
		return nil
	}
}
