package github

import (
	"context"
	"time"

	"github.com/shurcooL/githubv4"
)

type Discussion struct {
	Number    int                 `json:"number"`
	Title     string              `json:"title"`
	Body      string              `json:"body"`
	Author    Actor               `json:"author"`
	Category  string              `json:"category"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Comments  []DiscussionComment `json:"comments"`
}

type DiscussionComment struct {
	Author    Actor                    `json:"author"`
	Body      string                   `json:"body"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	Replies   []DiscussionCommentReply `json:"replies,omitempty"`
}

type DiscussionCommentReply struct {
	Author    Actor     `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type discussionQuery struct {
	Repository struct {
		Discussions struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []struct {
				Number    githubv4.Int
				Title     githubv4.String
				Body      githubv4.String
				CreatedAt githubv4.DateTime
				UpdatedAt githubv4.DateTime
				Author    struct {
					Login githubv4.String
				}
				Category struct {
					Name githubv4.String
				}
				Comments struct {
					Nodes []struct {
						Author struct {
							Login githubv4.String
						}
						Body      githubv4.String
						CreatedAt githubv4.DateTime
						UpdatedAt githubv4.DateTime
						Replies   struct {
							Nodes []struct {
								Author struct {
									Login githubv4.String
								}
								Body      githubv4.String
								CreatedAt githubv4.DateTime
								UpdatedAt githubv4.DateTime
							}
						} `graphql:"replies(first: 20)"`
					}
				} `graphql:"comments(first: 50)"`
			}
		} `graphql:"discussions(first: 10, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func (c *Client) FetchDiscussions(ctx context.Context, owner, repo string, since *time.Time) ([]Discussion, error) {
	var allDiscussions []Discussion
	var cursor *githubv4.String

	for {
		var q discussionQuery
		vars := map[string]any{
			"owner":  githubv4.String(owner),
			"repo":   githubv4.String(repo),
			"cursor": cursor,
		}

		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, err
		}

		reachedOldDiscussions := false
		for _, node := range q.Repository.Discussions.Nodes {
			// Since results are ordered by UPDATED_AT DESC, once we hit an old one, we're done
			if since != nil && node.UpdatedAt.Time.Before(*since) {
				reachedOldDiscussions = true
				break
			}

			disc := Discussion{
				Number:    int(node.Number),
				Title:     string(node.Title),
				Body:      string(node.Body),
				Author:    Actor{Login: string(node.Author.Login)},
				Category:  string(node.Category.Name),
				CreatedAt: node.CreatedAt.Time,
				UpdatedAt: node.UpdatedAt.Time,
			}

			for _, c := range node.Comments.Nodes {
				comment := DiscussionComment{
					Author:    Actor{Login: string(c.Author.Login)},
					Body:      string(c.Body),
					CreatedAt: c.CreatedAt.Time,
					UpdatedAt: c.UpdatedAt.Time,
				}

				for _, r := range c.Replies.Nodes {
					comment.Replies = append(comment.Replies, DiscussionCommentReply{
						Author:    Actor{Login: string(r.Author.Login)},
						Body:      string(r.Body),
						CreatedAt: r.CreatedAt.Time,
						UpdatedAt: r.UpdatedAt.Time,
					})
				}

				disc.Comments = append(disc.Comments, comment)
			}

			allDiscussions = append(allDiscussions, disc)
		}

		if reachedOldDiscussions || !q.Repository.Discussions.PageInfo.HasNextPage {
			break
		}
		cursor = &q.Repository.Discussions.PageInfo.EndCursor
	}

	return allDiscussions, nil
}
