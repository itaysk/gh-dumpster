package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/itaysk/gh-dumpster/internal/github"
	"github.com/itaysk/gh-dumpster/internal/storage"
)

type SyncOptions struct {
	Owner       string
	Repo        string
	OutputDir   string
	Issues      bool
	PRs         bool
	Discussions bool
	Since       *time.Time
}

func Sync(opts SyncOptions) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	store := storage.New(opts.OutputDir)
	if err := store.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create output directories: %w", err)
	}

	state, err := store.LoadSyncState()
	if err != nil {
		return fmt.Errorf("failed to load sync state: %w", err)
	}

	ctx := context.Background()
	syncTime := time.Now()

	// Use --since flag if provided, otherwise use stored state
	getSince := func(stored *time.Time) *time.Time {
		if opts.Since != nil {
			return opts.Since
		}
		return stored
	}

	if opts.Issues {
		if err := syncIssues(ctx, client, store, opts.Owner, opts.Repo, getSince(state.Issues)); err != nil {
			return fmt.Errorf("failed to sync issues: %w", err)
		}
		state.Issues = &syncTime
	}

	if opts.PRs {
		if err := syncPRs(ctx, client, store, opts.Owner, opts.Repo, getSince(state.PRs)); err != nil {
			return fmt.Errorf("failed to sync pull requests: %w", err)
		}
		state.PRs = &syncTime
	}

	if opts.Discussions {
		if err := syncDiscussions(ctx, client, store, opts.Owner, opts.Repo, getSince(state.Discussions)); err != nil {
			return fmt.Errorf("failed to sync discussions: %w", err)
		}
		state.Discussions = &syncTime
	}

	if err := store.SaveSyncState(state); err != nil {
		return fmt.Errorf("failed to save sync state: %w", err)
	}

	return nil
}

func syncIssues(ctx context.Context, client *github.Client, store *storage.Storage, owner, repo string, since *time.Time) error {
	fmt.Printf("Syncing issues from %s/%s", owner, repo)
	if since != nil {
		fmt.Printf(" (since %s)", since.Format(time.RFC3339))
	}
	fmt.Println()

	issues, err := client.FetchIssues(ctx, owner, repo, since)
	if err != nil {
		return err
	}

	fmt.Printf("  Found %d issues to sync\n", len(issues))
	for _, issue := range issues {
		if err := store.SaveIssue(issue.Number, issue); err != nil {
			return fmt.Errorf("failed to save issue %d: %w", issue.Number, err)
		}
	}
	return nil
}

func syncPRs(ctx context.Context, client *github.Client, store *storage.Storage, owner, repo string, since *time.Time) error {
	fmt.Printf("Syncing pull requests from %s/%s", owner, repo)
	if since != nil {
		fmt.Printf(" (since %s)", since.Format(time.RFC3339))
	}
	fmt.Println()

	prs, err := client.FetchPullRequests(ctx, owner, repo, since)
	if err != nil {
		return err
	}

	fmt.Printf("  Found %d pull requests to sync\n", len(prs))
	for _, pr := range prs {
		if err := store.SavePR(pr.Number, pr); err != nil {
			return fmt.Errorf("failed to save PR %d: %w", pr.Number, err)
		}
	}
	return nil
}

func syncDiscussions(ctx context.Context, client *github.Client, store *storage.Storage, owner, repo string, since *time.Time) error {
	fmt.Printf("Syncing discussions from %s/%s", owner, repo)
	if since != nil {
		fmt.Printf(" (since %s)", since.Format(time.RFC3339))
	}
	fmt.Println()

	discussions, err := client.FetchDiscussions(ctx, owner, repo, since)
	if err != nil {
		return err
	}

	fmt.Printf("  Found %d discussions to sync\n", len(discussions))
	for _, disc := range discussions {
		if err := store.SaveDiscussion(disc.Number, disc); err != nil {
			return fmt.Errorf("failed to save discussion %d: %w", disc.Number, err)
		}
	}
	return nil
}
