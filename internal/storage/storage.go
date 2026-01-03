package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SyncState struct {
	Issues      *time.Time `json:"issues,omitempty"`
	PRs         *time.Time `json:"prs,omitempty"`
	Discussions *time.Time `json:"discussions,omitempty"`
}

type Storage struct {
	baseDir string
}

func New(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

func (s *Storage) EnsureDirs() error {
	dirs := []string{
		filepath.Join(s.baseDir, "issues"),
		filepath.Join(s.baseDir, "pull_requests"),
		filepath.Join(s.baseDir, "discussions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) LoadSyncState() (*SyncState, error) {
	path := filepath.Join(s.baseDir, ".sync-state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SyncState{}, nil
	}
	if err != nil {
		return nil, err
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Storage) SaveSyncState(state *SyncState) error {
	path := filepath.Join(s.baseDir, ".sync-state.json")
	return s.atomicWrite(path, state)
}

func (s *Storage) SaveIssue(number int, data any) error {
	path := filepath.Join(s.baseDir, "issues", numberPrefix(number), formatNumber(number)+".json")
	return s.atomicWrite(path, data)
}

func (s *Storage) SavePR(number int, data any) error {
	path := filepath.Join(s.baseDir, "pull_requests", numberPrefix(number), formatNumber(number)+".json")
	return s.atomicWrite(path, data)
}

func (s *Storage) SaveDiscussion(number int, data any) error {
	path := filepath.Join(s.baseDir, "discussions", numberPrefix(number), formatNumber(number)+".json")
	return s.atomicWrite(path, data)
}

func (s *Storage) atomicWrite(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(jsonData); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

func formatNumber(n int) string {
	return fmt.Sprintf("%d", n)
}

func numberPrefix(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) < 2 {
		return s
	}
	return s[:2]
}
