package repository

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/crypto"
	"tuxedosql/pkg/fileutil"
)

const (
	connectionsFile = "connections.json"
	groupsFile      = "groups.json"
	keyFile         = ".key"
)

// ConnectionRepository 管理连接和分组的持久化存储。
// 密码字段以 AES-256-GCM 加密存储，密钥保存在 ~/.tuxedosql/.key。
type ConnectionRepository struct {
	mu    sync.RWMutex
	store *fileutil.JSONStore
	key   []byte // AES-256 密钥（32字节），首次使用时从文件加载或生成
}

// NewConnectionRepository 创建一个新的 ConnectionRepository。
func NewConnectionRepository(store *fileutil.JSONStore) *ConnectionRepository {
	return &ConnectionRepository{store: store}
}

// getOrCreateKey 获取或创建 AES-256 加密密钥。
// 密钥从 ~/.tuxedosql/.key 加载，如果不存在则自动生成。
func (r *ConnectionRepository) getOrCreateKey() ([]byte, error) {
	if r.key != nil {
		return r.key, nil
	}

	keyPath := filepath.Join(r.store.ConfigDir(), keyFile)

	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 生成新密钥
			keyHex, err := crypto.GenerateKey()
			if err != nil {
				return nil, fmt.Errorf("生成密钥失败: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
				return nil, fmt.Errorf("创建配置目录失败: %w", err)
			}
			if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
				return nil, fmt.Errorf("保存密钥文件失败: %w", err)
			}
			r.key, err = hex.DecodeString(keyHex)
			if err != nil {
				return nil, fmt.Errorf("解码密钥失败: %w", err)
			}
			return r.key, nil
		}
		return nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}

	keyHex := string(data)
	r.key, err = hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("解码密钥失败: %w", err)
	}
	if len(r.key) != 32 {
		return nil, fmt.Errorf("密钥长度不正确: %d 字节，期望 32", len(r.key))
	}
	return r.key, nil
}

// LoadConnections 从文件中加载所有连接，自动解密密码字段。
// 兼容旧版明文密码：检测到明文时自动加密迁移。
func (r *ConnectionRepository) LoadConnections() ([]model.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connections []model.Connection
	if err := r.store.Load(connectionsFile, &connections); err != nil {
		return nil, fmt.Errorf("加载连接: %w", err)
	}

	// 解密密码；兼容旧明文自动迁移
	needsMigration := false
	key, keyErr := r.getOrCreateKey()
	for i := range connections {
		if connections[i].Password == "" {
			continue
		}
		if crypto.IsEncrypted(connections[i].Password) {
			if keyErr != nil {
				continue // 密钥不可用，保留密文
			}
			ciphertext := connections[i].Password[len(crypto.EncryptedPrefix):]
			plaintext, err := crypto.Decrypt(ciphertext, key)
			if err != nil {
				connections[i].Password = "[无法解密]"
				continue
			}
			connections[i].Password = plaintext
		} else {
			needsMigration = true
		}
	}

	// 如果存在明文密码，迁移保存
	if needsMigration && keyErr == nil {
		_ = r.saveEncrypted(connections, key)
	}

	return connections, nil
}

// SaveConnections 将所有连接加密后保存到文件。
func (r *ConnectionRepository) SaveConnections(connections []model.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, err := r.getOrCreateKey()
	if err != nil {
		return fmt.Errorf("获取加密密钥失败: %w", err)
	}

	return r.saveEncrypted(connections, key)
}

// saveEncrypted 使用给定密钥加密密码后保存连接。
func (r *ConnectionRepository) saveEncrypted(connections []model.Connection, key []byte) error {
	encrypted := make([]model.Connection, len(connections))
	for i, c := range connections {
		encrypted[i] = c
		if c.Password != "" && !crypto.IsEncrypted(c.Password) {
			ciphertext, err := crypto.Encrypt(c.Password, key)
			if err != nil {
				return fmt.Errorf("加密密码失败: %w", err)
			}
			encrypted[i].Password = crypto.EncryptedPrefix + ciphertext
		}
	}

	if err := r.store.Save(connectionsFile, encrypted); err != nil {
		return fmt.Errorf("保存连接: %w", err)
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
