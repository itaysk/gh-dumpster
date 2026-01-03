package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/itaysk/gh-dumpster/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	outputDir string
	kinds     []string
	sinceStr  string
)

var rootCmd = &cobra.Command{
	Use:   "gh-dumpster",
	Short: "Track GitHub repository data locally",
}

var syncCmd = &cobra.Command{
	Use:   "sync owner/repo",
	Short: "Sync a GitHub repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format, expected owner/repo")
		}

		opts := tracker.SyncOptions{
			Owner:     parts[0],
			Repo:      parts[1],
			OutputDir: outputDir,
		}

		if sinceStr != "" {
			t, err := time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				t, err = time.Parse("2006-01-02", sinceStr)
				if err != nil {
					return fmt.Errorf("invalid --since format, use RFC3339 (2006-01-02T15:04:05Z) or date (2006-01-02)")
				}
			}
			opts.Since = &t
		}

		if len(kinds) == 0 {
			opts.Issues = true
			opts.PRs = true
			opts.Discussions = true
		} else {
			for _, k := range kinds {
				switch k {
				case "issue":
					opts.Issues = true
				case "pr":
					opts.PRs = true
				case "discussion":
					opts.Discussions = true
				default:
					return fmt.Errorf("unknown kind: %s (valid: issue, pr, discussion)", k)
				}
			}
		}

		return tracker.Sync(opts)
	},
}

func init() {
	syncCmd.Flags().StringVarP(&outputDir, "output", "o", "out", "Output directory for JSON files")
	syncCmd.Flags().StringSliceVarP(&kinds, "kinds", "k", nil, "Resource types to sync: issue, pr, discussion (default: all)")
	syncCmd.Flags().StringVar(&sinceStr, "since", "", "Sync items updated after this time (RFC3339 or YYYY-MM-DD)")
	rootCmd.AddCommand(syncCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
