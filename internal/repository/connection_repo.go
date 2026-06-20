package repository

import (
	"fmt"
	"log"
	"sync"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/credential"
	"tuxedosql/pkg/fileutil"
)

const (
	connectionsFile = "connections.json"
	groupsFile      = "groups.json"
)

// ConnectionRepository 管理连接和分组的持久化存储。
// 密码通过 credential.Manager 安全存储：优先使用 OS 密钥环，不可用时回退到机器 ID 派生的 AES 加密。
type ConnectionRepository struct {
	mu    sync.RWMutex
	store *fileutil.JSONStore
	cred  *credential.Manager
}

// NewConnectionRepository 创建一个新的 ConnectionRepository。
func NewConnectionRepository(store *fileutil.JSONStore) *ConnectionRepository {
	return &ConnectionRepository{
		store: store,
		cred:  credential.NewManager(store),
	}
}

// LoadConnections 从文件中加载所有连接，自动解密密码字段。
// 兼容三种历史格式：
// - "keyring:" 哨兵 → 从 OS 密钥环读取
// - "aes256gcm$" 密文 → 先尝试旧 .key 文件解密，再回退到机器 ID 密钥
// - 明文（旧版遗留） → 直接返回，下次保存时自动迁移
func (r *ConnectionRepository) LoadConnections() ([]model.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connections []model.Connection
	if err := r.store.Load(connectionsFile, &connections); err != nil {
		return nil, fmt.Errorf("加载连接: %w", err)
	}

	for i := range connections {
		if connections[i].Password == "" {
			continue
		}

		if credential.IsKeyringMarker(connections[i].Password) {
			pw, err := r.cred.Retrieve(connections[i].ID, connections[i].Password)
			if err != nil {
				connections[i].Password = "[密钥环不可用]"
				continue
			}
			connections[i].Password = pw
		} else if isLegacyAES(connections[i].Password) {
			// "aes256gcm$" 密文：先尝试旧 .key 文件，再回退机器 ID 密钥
			pw, err := r.cred.RetrieveWithLegacyKey(connections[i].ID, connections[i].Password)
			if err != nil {
				connections[i].Password = "[无法解密]"
				continue
			}
			connections[i].Password = pw
		}
		// 明文密码：直接返回，下次 SaveConnections 自动迁移
	}

	return connections, nil
}

// isLegacyAES 判断密码字段是否为 AES 加密密文（旧版或回退路径）。
func isLegacyAES(s string) bool {
	// 检查是否有 "aes256gcm$" 前缀，排除 "keyring:" 哨兵
	return len(s) > len("aes256gcm$") && s[:len("aes256gcm$")] == "aes256gcm$"
}

// SaveConnections 将所有连接安全存储后保存到文件。
// 明文密码通过 credential.Manager 存储：优先写入 OS 密钥环（标记为 "keyring:"），
// 不可用时回退到机器 ID 派生的 AES 加密（标记为 "aes256gcm$<hex>"）。
// 迁移完成后自动删除旧版 .key 文件。
func (r *ConnectionRepository) SaveConnections(connections []model.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查旧版 .key 文件是否存在（迁移前记录）
	hadLegacyKey := false
	legacyKey, _ := r.cred.LoadLegacyKey()
	if legacyKey != nil {
		hadLegacyKey = true
	}

	saved := make([]model.Connection, len(connections))
	for i, c := range connections {
		saved[i] = c

		// 跳过：空密码、已存储在密钥环、已 AES 加密
		if c.Password == "" || credential.IsKeyringMarker(c.Password) || isLegacyAES(c.Password) {
			continue
		}

		// 明文密码：通过凭证管理器安全存储
		storedPw, _, err := r.cred.Store(c.ID, c.Password)
		if err != nil {
			return fmt.Errorf("存储密码失败 (连接 %s): %w", c.ID, err)
		}
		saved[i].Password = storedPw
	}

	if err := r.store.Save(connectionsFile, saved); err != nil {
		return fmt.Errorf("保存连接: %w", err)
	}

	// 迁移完成：所有密码已通过 keyring 或机器 ID 密钥存储，旧 .key 文件不再需要
	if hadLegacyKey {
		if err := r.cred.DeleteLegacyKey(); err != nil {
			log.Printf("删除旧密钥文件失败 (下次启动时会再次尝试): %v", err)
		} else {
			log.Printf("旧密钥文件已成功删除，密码迁移完成")
		}
	}

	return nil
}

// LoadGroups 从文件中加载所有分组。
func (r *ConnectionRepository) LoadGroups() ([]model.ConnectionGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var groups []model.ConnectionGroup
	if err := r.store.Load(groupsFile, &groups); err != nil {
		return nil, fmt.Errorf("加载分组: %w", err)
	}
	return groups, nil
}

// SaveGroups 将所有分组保存到文件。
func (r *ConnectionRepository) SaveGroups(groups []model.ConnectionGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.Save(groupsFile, groups); err != nil {
		return fmt.Errorf("保存分组: %w", err)
	}
	return nil
}

// LoadConnectionByID 根据ID加载单个连接配置（含密码解密）。
func (r *ConnectionRepository) LoadConnectionByID(id string) (*model.Connection, error) {
	connections, err := r.LoadConnections()
	if err != nil {
		return nil, fmt.Errorf("加载连接: %w", err)
	}
	for i := range connections {
		if connections[i].ID == id {
			return &connections[i], nil
		}
	}
	return nil, fmt.Errorf("连接不存在: %s", id)
}

// DeleteCredential 删除指定连接在 OS 密钥环中的凭证条目。
// 在删除连接后调用，确保密钥环不留残余数据。
func (r *ConnectionRepository) DeleteCredential(connectionID string) {
	_ = r.cred.Delete(connectionID)
}
