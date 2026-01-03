package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
)

type PullRequest struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Author    Actor      `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	Labels    []Label    `json:"labels"`
	Comments  []Comment  `json:"comments"`
	Reviews   []Review   `json:"reviews"`
	Events    []Event    `json:"events"`
}

type ReviewComment struct {
	Author    Actor     `json:"author"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

type Review struct {
	Author      Actor           `json:"author"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	SubmittedAt time.Time       `json:"submitted_at"`
	Comments    []ReviewComment `json:"comments,omitempty"`
}

type prQuery struct {
	Repository struct {
		PullRequests struct {
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
				MergedAt  *githubv4.DateTime
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
				Reviews struct {
					Nodes []struct {
						Author struct {
							Login githubv4.String
						}
						Body        githubv4.String
						State       githubv4.String
						SubmittedAt *githubv4.DateTime
						Comments    struct {
							Nodes []struct {
								Author struct {
									Login githubv4.String
								}
								Body      githubv4.String
								Path      githubv4.String
								CreatedAt githubv4.DateTime
							}
						} `graphql:"comments(first: 50)"`
					}
				} `graphql:"reviews(first: 50)"`
				TimelineItems struct {
					Nodes []prTimelineItem
				} `graphql:"timelineItems(first: 50)"`
			}
		} `graphql:"pullRequests(first: 20, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

type mergedEvent struct {
	Actor     struct{ Login githubv4.String }
	CreatedAt githubv4.DateTime
}

type reviewRequestedEvent struct {
	Actor            struct{ Login githubv4.String }
	CreatedAt        githubv4.DateTime
	RequestedReviewer struct {
		User struct{ Login githubv4.String } `graphql:"... on User"`
	}
}

type pullRequestCommit struct {
	Commit struct {
		Oid     githubv4.String
		Message githubv4.String
		Author  struct {
			Name githubv4.String
		}
	}
}

type prTimelineItem struct {
	TypeName             string               `graphql:"__typename"`
	ClosedEvent          closedEvent          `graphql:"... on ClosedEvent"`
	ReopenedEvent        reopenedEvent        `graphql:"... on ReopenedEvent"`
	MergedEvent          mergedEvent          `graphql:"... on MergedEvent"`
	LabeledEvent         labeledEvent         `graphql:"... on LabeledEvent"`
	UnlabeledEvent       unlabeledEvent       `graphql:"... on UnlabeledEvent"`
	AssignedEvent        assignedEvent        `graphql:"... on AssignedEvent"`
	ReviewRequestedEvent reviewRequestedEvent `graphql:"... on ReviewRequestedEvent"`
	PullRequestCommit    pullRequestCommit    `graphql:"... on PullRequestCommit"`
}

func (c *Client) FetchPullRequests(ctx context.Context, owner, repo string, since *time.Time) ([]PullRequest, error) {
	var allPRs []PullRequest
	var cursor *githubv4.String

	for {
		var q prQuery
		vars := map[string]any{
			"owner":  githubv4.String(owner),
			"repo":   githubv4.String(repo),
			"cursor": cursor,
		}

		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, err
		}

		for _, node := range q.Repository.PullRequests.Nodes {
			// Skip PRs not updated since the last sync
			if since != nil && node.UpdatedAt.Time.Before(*since) {
				continue
			}

			pr := PullRequest{
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
				pr.ClosedAt = &t
			}
			if node.MergedAt != nil {
				t := node.MergedAt.Time
				pr.MergedAt = &t
			}

			for _, l := range node.Labels.Nodes {
				pr.Labels = append(pr.Labels, Label{
					Name:  string(l.Name),
					Color: string(l.Color),
				})
			}

			for _, c := range node.Comments.Nodes {
				pr.Comments = append(pr.Comments, Comment{
					Author:    Actor{Login: string(c.Author.Login)},
					Body:      string(c.Body),
					CreatedAt: c.CreatedAt.Time,
					UpdatedAt: c.UpdatedAt.Time,
				})
			}

			for _, r := range node.Reviews.Nodes {
				review := Review{
					Author: Actor{Login: string(r.Author.Login)},
					Body:   string(r.Body),
					State:  string(r.State),
				}
				if r.SubmittedAt != nil {
					review.SubmittedAt = r.SubmittedAt.Time
				}
				for _, c := range r.Comments.Nodes {
					review.Comments = append(review.Comments, ReviewComment{
						Author:    Actor{Login: string(c.Author.Login)},
						Body:      string(c.Body),
						Path:      string(c.Path),
						CreatedAt: c.CreatedAt.Time,
					})
				}
				pr.Reviews = append(pr.Reviews, review)
			}

			for _, ti := range node.TimelineItems.Nodes {
				event := convertPRTimelineEvent(ti)
				if event != nil {
					pr.Events = append(pr.Events, *event)
				}
			}

			allPRs = append(allPRs, pr)
		}

		if !q.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		// If we got items older than since, we can stop
		if since != nil && len(q.Repository.PullRequests.Nodes) > 0 {
			lastNode := q.Repository.PullRequests.Nodes[len(q.Repository.PullRequests.Nodes)-1]
			if lastNode.UpdatedAt.Time.Before(*since) {
				break
			}
		}
		cursor = &q.Repository.PullRequests.PageInfo.EndCursor
	}

	return allPRs, nil
}

func convertPRTimelineEvent(ti prTimelineItem) *Event {
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
	case "MergedEvent":
		return &Event{
			Type:      "merged",
			Actor:     Actor{Login: string(ti.MergedEvent.Actor.Login)},
			CreatedAt: ti.MergedEvent.CreatedAt.Time,
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
	case "ReviewRequestedEvent":
		return &Event{
			Type:      "review_requested",
			Actor:     Actor{Login: string(ti.ReviewRequestedEvent.Actor.Login)},
			CreatedAt: ti.ReviewRequestedEvent.CreatedAt.Time,
			Details:   map[string]string{"reviewer": string(ti.ReviewRequestedEvent.RequestedReviewer.User.Login)},
		}
	case "PullRequestCommit":
		return &Event{
			Type:      "commit",
			Actor:     Actor{Login: string(ti.PullRequestCommit.Commit.Author.Name)},
			Details: map[string]string{
				"sha":     string(ti.PullRequestCommit.Commit.Oid),
				"message": string(ti.PullRequestCommit.Commit.Message),
			},
		}
	default:
		return nil
	}
}
