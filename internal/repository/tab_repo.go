package repository

import (
	"fmt"
	"sync"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/fileutil"
)

const tabsFile = "tabs.json"

// TabRepository 管理查询标签页状态的持久化存储。
type TabRepository struct {
	mu    sync.RWMutex
	store *fileutil.JSONStore
}

// NewTabRepository 创建一个新的 TabRepository。
func NewTabRepository(store *fileutil.JSONStore) *TabRepository {
	return &TabRepository{store: store}
}

// LoadTabs 从文件中加载所有标签页状态。
func (r *TabRepository) LoadTabs() ([]model.TabState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tabs []model.TabState
	if err := r.store.Load(tabsFile, &tabs); err != nil {
		return nil, fmt.Errorf("加载标签页: %w", err)
	}
	return tabs, nil
}

// SaveTabs 将所有标签页状态保存到文件。
func (r *TabRepository) SaveTabs(tabs []model.TabState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.Save(tabsFile, tabs); err != nil {
		return fmt.Errorf("保存标签页: %w", err)
	}
	return nil
}
