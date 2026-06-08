// Package credential 提供密码安全存储的抽象层，优先使用 OS 原生密钥环，
// 不可用时回退到基于机器 ID 派生的 AES-256-GCM 加密。
package credential

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/denisbrodbeck/machineid"
	keyring "github.com/zalando/go-keyring"
	"golang.org/x/crypto/pbkdf2"

	"tuxedosql/pkg/crypto"
	"tuxedosql/pkg/fileutil"
)

const (
	keyringService   = "tuxedosql"
	keyringKeyPrefix = "conn_"
	machineIDSalt    = "tuxedosql-app-key"
	pbkdf2Iterations = 100000
	legacyKeyFile    = ".key"
)

// StorageType 表示当前使用的凭证存储类型。
type StorageType string

const (
	StorageKeyring     StorageType = "keyring"
	StorageAESFallback StorageType = "aes-fallback"
)

// KeyringPrefix 是 JSON 中标记密码存储在密钥环的哨兵值。
const KeyringPrefix = "keyring:"

// IsKeyringMarker 判断密码字段是否为密钥环哨兵标记。
func IsKeyringMarker(s string) bool {
	return s == KeyringPrefix
}

// Manager 管理凭证的安全存储与检索。
// 优先使用 OS 原生密钥环（macOS Keychain / Linux libsecret / Windows Credential Manager），
// 密钥环不可用时回退到从机器 ID 派生的 AES-256-GCM 加密。
type Manager struct {
	configDir  string // 配置目录路径（存放 .key 等文件）
	machineKey []byte // 从机器 ID 派生的 AES 密钥（懒加载并缓存）
	oldKey     []byte // 旧版 .key 文件的 AES 密钥（懒加载，仅迁移期使用）
	keyOnce    sync.Once
	oldKeyOnce sync.Once
	oldKeyErr  error
}

// NewManager 创建一个新的凭证管理器，使用 JSONStore 的配置目录。
func NewManager(store *fileutil.JSONStore) *Manager {
	return &Manager{configDir: store.ConfigDir()}
}

// newManagerWithDir 创建一个使用指定配置目录的凭证管理器（仅用于测试）。
func newManagerWithDir(configDir string) *Manager {
	return &Manager{configDir: configDir}
}

// Store 安全存储密码。优先使用 OS 密钥环，失败时回退到 AES 加密。
// 返回写入 connections.json 的 Password 字段值和使用的存储类型。
func (m *Manager) Store(connectionID, password string) (string, StorageType, error) {
	if password == "" {
		return "", StorageKeyring, nil
	}

	// 优先尝试 OS 密钥环
	key := keyringKeyPrefix + connectionID
	err := keyring.Set(keyringService, key, password)
	if err == nil {
		return KeyringPrefix, StorageKeyring, nil
	}

	// 密钥环不可用，回退到 AES 加密
	log.Printf("密钥环不可用 (%v)，回退到 AES 加密", err)
	aesKey, deriveErr := m.deriveKeyFromMachineID()
	if deriveErr != nil {
		return "", StorageAESFallback, fmt.Errorf("密钥环和 AES 回退均失败: 密钥环=%w, 派生=%v", err, deriveErr)
	}

	ciphertext, encErr := crypto.Encrypt(password, aesKey)
	if encErr != nil {
		return "", StorageAESFallback, fmt.Errorf("AES 加密失败: %w", encErr)
	}

	return crypto.EncryptedPrefix + ciphertext, StorageAESFallback, nil
}

// Retrieve 检索密码。根据存储标记决定读取路径：
// - "keyring:" → 从 OS 密钥环读取
// - "aes256gcm$" → 使用机器 ID 派生的 AES 密钥解密
// - 明文 → 直接返回（旧版遗留）
func (m *Manager) Retrieve(connectionID, storedPassword string) (string, error) {
	if storedPassword == "" {
		return "", nil
	}

	// 密钥环标记：从 OS 密钥环读取
	if IsKeyringMarker(storedPassword) {
		key := keyringKeyPrefix + connectionID
		password, err := keyring.Get(keyringService, key)
		if err != nil {
			return "", fmt.Errorf("从密钥环读取密码失败 (连接 %s): %w", connectionID, err)
		}
		return password, nil
	}

	// AES 加密密文：使用机器 ID 派生的密钥解密
	if crypto.IsEncrypted(storedPassword) {
		aesKey, err := m.deriveKeyFromMachineID()
		if err != nil {
			return "", fmt.Errorf("派生 AES 密钥失败: %w", err)
		}
		ciphertext := storedPassword[len(crypto.EncryptedPrefix):]
		plaintext, err := crypto.Decrypt(ciphertext, aesKey)
		if err != nil {
			return "", fmt.Errorf("AES 解密失败 (连接 %s): %w", connectionID, err)
		}
		return plaintext, nil
	}

	// 明文（旧版遗留），直接返回
	return storedPassword, nil
}

// RetrieveWithLegacyKey 迁移期专用：先尝试旧 .key 文件解密，再回退到机器 ID 密钥。
// 用于从旧版（.key 文件加密）到新版（密钥环 / 机器 ID 密钥）的平滑迁移。
func (m *Manager) RetrieveWithLegacyKey(connectionID, storedPassword string) (string, error) {
	if !crypto.IsEncrypted(storedPassword) {
		return m.Retrieve(connectionID, storedPassword)
	}

	// 尝试旧版 .key 文件
	legacyKey, loadErr := m.LoadLegacyKey()
	if loadErr != nil {
		// .key 文件读取出错（不是"文件不存在"），回退到机器 ID
		return m.Retrieve(connectionID, storedPassword)
	}
	if legacyKey != nil {
		ciphertext := storedPassword[len(crypto.EncryptedPrefix):]
		plaintext, decErr := crypto.Decrypt(ciphertext, legacyKey)
		if decErr == nil {
			return plaintext, nil
		}
		// 旧密钥解密失败（可能已被新密钥重新加密），回退到机器 ID
	}

	// 无旧密钥或旧密钥解密失败：使用机器 ID 派生的密钥
	return m.Retrieve(connectionID, storedPassword)
}

// Delete 删除存储的密码。优先从密钥环删除，忽略"不存在"错误。
func (m *Manager) Delete(connectionID string) error {
	key := keyringKeyPrefix + connectionID
	_ = keyring.Delete(keyringService, key)
	return nil
}

// CurrentStorageType 返回当前可用的存储类型（用于诊断信息）。
func (m *Manager) CurrentStorageType() StorageType {
	testKey := keyringKeyPrefix + "__availability_test__"
	err := keyring.Set(keyringService, testKey, "test")
	if err != nil {
		return StorageAESFallback
	}
	_ = keyring.Delete(keyringService, testKey)
	return StorageKeyring
}

// deriveKeyFromMachineID 使用 PBKDF2 从机器 ID 派生 AES-256 密钥。
// machineid.ProtectedID 生成绑定于 OS 安装的唯一标识，
// 再通过 PBKDF2-SHA256 + 固定 salt 扩展为 32 字节 AES 密钥。
func (m *Manager) deriveKeyFromMachineID() ([]byte, error) {
	m.keyOnce.Do(func() {
		machineID, err := machineid.ProtectedID(machineIDSalt)
		if err != nil {
			m.machineKey = nil
			m.keyOnce = sync.Once{} // 重置，允许下次重试
			return
		}
		m.machineKey = pbkdf2.Key([]byte(machineID), []byte(machineIDSalt), pbkdf2Iterations, 32, sha256.New)
	})

	if m.machineKey == nil {
		return nil, fmt.Errorf("获取机器 ID 失败")
	}
	return m.machineKey, nil
}

// LoadLegacyKey 加载旧版 .key 文件中的 AES 密钥，用于迁移兼容。
// 返回 nil 表示文件不存在（已迁移完成），不需要旧密钥。
func (m *Manager) LoadLegacyKey() ([]byte, error) {
	m.oldKeyOnce.Do(func() {
		keyPath := filepath.Join(m.configDir, legacyKeyFile)
		data, err := os.ReadFile(keyPath)
		if err != nil {
			if os.IsNotExist(err) {
				m.oldKey = nil
				return
			}
			m.oldKeyErr = fmt.Errorf("读取旧密钥文件失败: %w", err)
			return
		}

		keyHex := string(data)
		decoded, err := hex.DecodeString(keyHex)
		if err != nil {
			m.oldKeyErr = fmt.Errorf("解码旧密钥失败: %w", err)
			return
		}
		if len(decoded) != 32 {
			m.oldKeyErr = fmt.Errorf("旧密钥长度不正确: %d 字节，期望 32", len(decoded))
			return
		}
		m.oldKey = decoded
	})

	if m.oldKeyErr != nil {
		return nil, m.oldKeyErr
	}
	return m.oldKey, nil
}

// DeleteLegacyKey 删除旧版 .key 文件。迁移完成后调用。
func (m *Manager) DeleteLegacyKey() error {
	keyPath := filepath.Join(m.configDir, legacyKeyFile)
	err := os.Remove(keyPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除旧密钥文件失败: %w", err)
	}
	// 清除缓存，下次 LoadLegacyKey 返回 nil
	m.oldKey = nil
	m.oldKeyOnce = sync.Once{}
	return nil
}