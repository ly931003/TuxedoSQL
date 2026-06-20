package repository

import (
	"fmt"
	"sync"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/fileutil"
)

const (
	historyFile       = "history.json"
	maxHistoryEntries = 200
)

type HistoryRepository struct {
	mu    sync.RWMutex
	store *fileutil.JSONStore
}

func NewHistoryRepository(store *fileutil.JSONStore) *HistoryRepository {
	return &HistoryRepository{store: store}
}

func (r *HistoryRepository) LoadHistory() ([]model.QueryHistoryEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var entries []model.QueryHistoryEntry
	if err := r.store.Load(historyFile, &entries); err != nil {
		return nil, fmt.Errorf("加载查询历史: %w", err)
	}
	if entries == nil {
		return []model.QueryHistoryEntry{}, nil
	}

	return entries, nil
}

func (r *HistoryRepository) SaveHistory(entries []model.QueryHistoryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entries == nil {
		entries = []model.QueryHistoryEntry{}
	}
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}

	if err := r.store.Save(historyFile, entries); err != nil {
		return fmt.Errorf("保存查询历史: %w", err)
	}

	return nil
}
