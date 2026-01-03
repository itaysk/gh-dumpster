# gh-dumpster

Track GitHub Issues, Pull Requests, and Discussions as local files.

## Installation

```bash
go install github.com/itaysk/gh-dumpster@latest
```

## Usage

```bash
export GITHUB_TOKEN=your_token

# Sync all resources from a repository
gh-dumpster sync owner/repo

# Specify output directory (default: ./out)
gh-dumpster sync owner/repo --output ./my-data

# Sync specific resource types only
gh-dumpster sync owner/repo --kinds issue,pr  # skip discussions
gh-dumpster sync owner/repo -k issue -k pr    # alternative syntax

# Sync items updated after a specific date/time
gh-dumpster sync owner/repo --since 2024-01-01
gh-dumpster sync owner/repo --since 2024-01-15T10:30:00Z
```

## Data Storage Format

```
out/
  issues/
    12/
      123.json          # Issue with comments and events
  pull_requests/
    45/
      456.json          # PR with comments, reviews, events
  discussions/
    78/
      789.json          # Discussion with comments
  .sync-state.json    # Tracks last sync timestamps
```

Each JSON file contains the full item data fields, events, comments, etc.

## Incremental Sync

The tool tracks the last sync timestamp per resource type in `.sync-state.json`. On subsequent runs, it only fetches items updated since the last sync, making it efficient for periodic syncing.
Use `--since` to override the stored timestamp and sync from a specific point in time. Accepts RFC3339 (`2024-01-15T10:30:00Z`) or date (`2024-01-15`) format.

## Failure Resilience

- **Atomic writes**: Files are written to a temp location first, then renamed to the target path (prevents corrupted files on crash)
- **Per-resource state**: Sync state is tracked per resource type, so partial failures don't require a full re-sync
- **Idempotent**: Re-running after a failure picks up where it left off

## Authentication

GitHub Personal Access Token with `repo` scope (for private repos) or `public_repo` scope (for public repos only)
