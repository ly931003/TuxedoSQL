package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"tuxedosql/internal/model"
)

// ConnectionManager 管理 MySQL 连接池，按 connectionID:database 缓存 *sql.DB 实例。
// 不同数据库使用独立的连接池，避免 USE database 交叉影响。
type ConnectionManager struct {
	mu       sync.RWMutex
	pools    map[string]*sql.DB // key: connectionID:database
	connRepo *ConnectionRepository
}

// NewConnectionManager 创建一个新的 ConnectionManager。
func NewConnectionManager(connRepo *ConnectionRepository) *ConnectionManager {
	return &ConnectionManager{
		pools:    make(map[string]*sql.DB),
		connRepo: connRepo,
	}
}

// GetDB 返回指定连接和数据库对应的池化 *sql.DB。
// 不同数据库使用独立的连接池（key = connectionID:database），
// 每个池在创建时 DSN 即指定 database，后续无需 USE 切换。
func (m *ConnectionManager) GetDB(conn *model.Connection, database string) (*sql.DB, error) {
	if conn == nil {
		return nil, fmt.Errorf("连接不能为空")
	}

	poolKey := conn.ID + ":" + database

	// 快速路径：读锁检查是否已存在
	m.mu.RLock()
	if db, ok := m.pools[poolKey]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	// 慢速路径：写锁创建新连接池
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查：可能在等待写锁期间已被其他 goroutine 创建
	if db, ok := m.pools[poolKey]; ok {
		return db, nil
	}

	// DSN 指定数据库，每个 database 独立的连接池
	dbName := database
	if dbName == "" {
		dbName = conn.Database
	}
	if dbName == "" {
		dbName = "mysql"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&parseTime=true",
		conn.Username, conn.Password, conn.Host, conn.Port, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 配置连接池参数
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(30 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)

	// 首次连接验证
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接测试失败: %w", err)
	}

	m.pools[poolKey] = db
	return db, nil
}

// GetDBByID 根据连接ID和数据库名获取池化连接。
// 内部从 ConnectionRepository 加载连接配置后调用 GetDB。
func (m *ConnectionManager) GetDBByID(connectionID, database string) (*model.Connection, *sql.DB, error) {
	conn, err := m.connRepo.LoadConnectionByID(connectionID)
	if err != nil {
		return nil, nil, err
	}

	db, err := m.GetDB(conn, database)
	if err != nil {
		return conn, nil, err
	}

	return conn, db, nil
}

// Close 关闭并移除指定连接的所有连接池（匹配 connectionID: 前缀的所有 key）。
// pool key 格式为 "connectionID:database"，因此需要前缀匹配而非精确查找。
func (m *ConnectionManager) Close(connectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefix := connectionID + ":"
	for key, db := range m.pools {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			db.Close()
			delete(m.pools, key)
		}
	}
}

// CloseAll 关闭所有连接池。通常在应用退出时调用。
func (m *ConnectionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, db := range m.pools {
		db.Close()
		delete(m.pools, id)
	}
}
