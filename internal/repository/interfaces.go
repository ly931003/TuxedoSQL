package repository

import (
	"database/sql"

	"tuxedosql/internal/model"
)

type ConnectionStore interface {
	LoadConnections() ([]model.Connection, error)
	SaveConnections(connections []model.Connection) error
	LoadGroups() ([]model.ConnectionGroup, error)
	SaveGroups(groups []model.ConnectionGroup) error
	LoadConnectionByID(id string) (*model.Connection, error)
	DeleteCredential(connectionID string)
}

type PoolManager interface {
	GetDB(conn *model.Connection, database string) (*sql.DB, error)
	GetDBByID(connectionID, database string) (*model.Connection, *sql.DB, error)
	Close(connectionID string)
}

type TabStore interface {
	LoadTabs() ([]model.TabState, error)
	SaveTabs(tabs []model.TabState) error
}

type HistoryStore interface {
	LoadHistory() ([]model.QueryHistoryEntry, error)
	SaveHistory(entries []model.QueryHistoryEntry) error
}

var (
	_ ConnectionStore = (*ConnectionRepository)(nil)
	_ PoolManager     = (*ConnectionManager)(nil)
	_ TabStore        = (*TabRepository)(nil)
	_ HistoryStore    = (*HistoryRepository)(nil)
)
