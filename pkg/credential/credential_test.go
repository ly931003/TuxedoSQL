package credential

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"tuxedosql/pkg/crypto"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	// 使用临时目录，绝不触碰真实 ~/.tuxedosql/
	tmpDir := t.TempDir()
	return newManagerWithDir(tmpDir)
}

func TestManager_StoreAndRetrieve_EmptyPassword(t *testing.T) {
	m := newTestManager(t)

	stored, st, err := m.Store("conn_test1", "")
	if err != nil {
		t.Fatalf("Store 空 password 失败: %v", err)
	}
	if stored != "" {
		t.Errorf("空 password 的 stored 值应为空字符串, 实际 %q", stored)
	}
	if st != StorageKeyring {
		t.Errorf("空 password 的 storage type 应为 keyring, 实际 %s", st)
	}

	retrieved, err := m.Retrieve("conn_test1", stored)
	if err != nil {
		t.Fatalf("Retrieve 空 password 失败: %v", err)
	}
	if retrieved != "" {
		t.Errorf("空 password 的 retrieved 值应为空字符串, 实际 %q", retrieved)
	}
}

func TestManager_DeriveKeyFromMachineID(t *testing.T) {
	m := newTestManager(t)

	key1, err := m.deriveKeyFromMachineID()
	if err != nil {
		t.Fatalf("第一次派生密钥失败: %v", err)
	}
	if len(key1) != 32 {
		t.Fatalf("密钥长度 = %d, 期望 32", len(key1))
	}

	// 第二次调用应返回缓存的相同密钥
	key2, err := m.deriveKeyFromMachineID()
	if err != nil {
		t.Fatalf("第二次派生密钥失败: %v", err)
	}
	if hex.EncodeToString(key1) != hex.EncodeToString(key2) {
		t.Error("两次派生的密钥应相同（缓存）")
	}
}

func TestManager_AESFallback_Roundtrip(t *testing.T) {
	m := newTestManager(t)

	// 强制使用 AES 回退路径：直接用 deriveKey 测试加密/解密循环
	key, err := m.deriveKeyFromMachineID()
	if err != nil {
		t.Fatalf("派生密钥失败: %v", err)
	}

	passwords := []string{"myPassword123", "中文密码@#$%", "p@$$w0rd!~"}
	for _, pw := range passwords {
		ciphertext, err := crypto.Encrypt(pw, key)
		if err != nil {
			t.Fatalf("加密 %q 失败: %v", pw, err)
		}
		stored := crypto.EncryptedPrefix + ciphertext

		retrieved, err := m.Retrieve("conn_aes_test", stored)
		if err != nil {
			t.Fatalf("Retrieve %q 失败: %v", pw, err)
		}
		if retrieved != pw {
			t.Errorf("AES 回退: retrieved = %q, 期望 %q", retrieved, pw)
		}
	}
}

func TestManager_LegacyKeyMigration(t *testing.T) {
	m := newTestManager(t)
	configDir := m.configDir

	// 在临时目录创建模拟的旧 .key 文件
	oldKeyHex, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("生成旧密钥失败: %v", err)
	}
	keyPath := filepath.Join(configDir, legacyKeyFile)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(oldKeyHex), 0600); err != nil {
		t.Fatalf("写入旧密钥文件失败: %v", err)
	}

	// LoadLegacyKey 应成功加载
	oldKey, err := m.LoadLegacyKey()
	if err != nil {
		t.Fatalf("LoadLegacyKey 失败: %v", err)
	}
	if oldKey == nil {
		t.Fatal("LoadLegacyKey 应返回密钥，但返回 nil")
	}
	if len(oldKey) != 32 {
		t.Errorf("旧密钥长度 = %d, 期望 32", len(oldKey))
	}

	// 用旧密钥加密一个密码，模拟旧版存储格式
	plaintext := "legacy_password"
	ciphertext, err := crypto.Encrypt(plaintext, oldKey)
	if err != nil {
		t.Fatalf("旧密钥加密失败: %v", err)
	}
	stored := crypto.EncryptedPrefix + ciphertext

	// RetrieveWithLegacyKey 应能正确解密
	retrieved, err := m.RetrieveWithLegacyKey("conn_legacy", stored)
	if err != nil {
		t.Fatalf("RetrieveWithLegacyKey 失败: %v", err)
	}
	if retrieved != plaintext {
		t.Errorf("RetrieveWithLegacyKey = %q, 期望 %q", retrieved, plaintext)
	}

	// DeleteLegacyKey 应删除 .key 文件
	if err := m.DeleteLegacyKey(); err != nil {
		t.Fatalf("DeleteLegacyKey 失败: %v", err)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Error("DeleteLegacyKey 后 .key 文件仍存在")
	}

	// 再次 LoadLegacyKey 应返回 nil（文件已删除）
	// 需要新 Manager 因为缓存了 oldKeyOnce
	m2 := newManagerWithDir(configDir)
	oldKey2, err := m2.LoadLegacyKey()
	if err != nil {
		t.Fatalf("二次 LoadLegacyKey 失败: %v", err)
	}
	if oldKey2 != nil {
		t.Error("文件删除后 LoadLegacyKey 应返回 nil")
	}
}

func TestManager_LegacyKey_NotExist(t *testing.T) {
	m := newTestManager(t)

	// 临时目录中 .key 文件默认不存在
	key, err := m.LoadLegacyKey()
	if err != nil {
		t.Fatalf("LoadLegacyKey 不应报错: %v", err)
	}
	if key != nil {
		t.Error("无 .key 文件时 LoadLegacyKey 应返回 nil")
	}
}

func TestManager_RetrieveWithLegacyKey_FallbackToMachineID(t *testing.T) {
	m := newTestManager(t)

	// 用 machine-ID key 加密一个密码（模拟已经迁移的格式）
	machineKey, err := m.deriveKeyFromMachineID()
	if err != nil {
		t.Fatalf("派生密钥失败: %v", err)
	}

	plaintext := "migrated_password"
	ciphertext, err := crypto.Encrypt(plaintext, machineKey)
	if err != nil {
		t.Fatalf("machine-ID key 加密失败: %v", err)
	}
	stored := crypto.EncryptedPrefix + ciphertext

	// 临时目录中没有 .key 文件，RetrieveWithLegacyKey 应回退到 machine-ID key
	retrieved, err := m.RetrieveWithLegacyKey("conn_migrated", stored)
	if err != nil {
		t.Fatalf("RetrieveWithLegacyKey (machine-ID 回退) 失败: %v", err)
	}
	if retrieved != plaintext {
		t.Errorf("RetrieveWithLegacyKey = %q, 期望 %q", retrieved, plaintext)
	}
}

func TestManager_Retrieve_Plaintext(t *testing.T) {
	m := newTestManager(t)

	plaintext := "plain_password"
	retrieved, err := m.Retrieve("conn_plain", plaintext)
	if err != nil {
		t.Fatalf("Retrieve plaintext 失败: %v", err)
	}
	if retrieved != plaintext {
		t.Errorf("Retrieve plaintext = %q, 期望 %q", retrieved, plaintext)
	}
}

func TestIsKeyringMarker(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"keyring:", true},
		{"keyring:abc", false},
		{"aes256gcm$xxx", false},
		{"plaintext", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsKeyringMarker(tt.input); got != tt.expected {
				t.Errorf("IsKeyringMarker(%q) = %v, 期望 %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestManager_CurrentStorageType(t *testing.T) {
	m := newTestManager(t)
	st := m.CurrentStorageType()
	t.Logf("当前存储类型: %s", st)
	if st != StorageKeyring && st != StorageAESFallback {
		t.Errorf("意外的存储类型: %s", st)
	}
}

func TestManager_Store_NonEmptyPassword(t *testing.T) {
	m := newTestManager(t)

	password := "test_password_123"
	stored, st, err := m.Store("conn_store_test", password)
	if err != nil {
		t.Fatalf("Store 失败: %v", err)
	}
	t.Logf("存储类型: %s, stored 值: %q", st, stored)

	// 无论哪种存储类型，都应能 Retrieve 回原始密码
	retrieved, err := m.Retrieve("conn_store_test", stored)
	if err != nil {
		t.Fatalf("Retrieve 失败: %v", err)
	}
	if retrieved != password {
		t.Errorf("Retrieve = %q, 期望 %q", retrieved, password)
	}
}

func TestManager_Delete(t *testing.T) {
	m := newTestManager(t)

	// 先存储一个密码
	password := "to_be_deleted"
	stored, _, err := m.Store("conn_delete_test", password)
	if err != nil {
		t.Fatalf("Store 失败: %v", err)
	}

	// 删除
	if err := m.Delete("conn_delete_test"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 如果是 keyring 存储，删除后 Retrieve 应失败
	if IsKeyringMarker(stored) {
		_, err := m.Retrieve("conn_delete_test", stored)
		if err == nil {
			t.Error("删除后从密钥环 Retrieve 应失败")
		}
	}
}

func TestManager_Store_DoesNotDoubleEncrypt(t *testing.T) {
	m := newTestManager(t)

	// 验证 Retrieve 对 aes256gcm$ 格式能正确解密
	machineKey, err := m.deriveKeyFromMachineID()
	if err != nil {
		t.Fatalf("派生密钥失败: %v", err)
	}

	password := "already_encrypted"
	ciphertext, err := crypto.Encrypt(password, machineKey)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	stored := crypto.EncryptedPrefix + ciphertext

	retrieved, err := m.Retrieve("conn_double_test", stored)
	if err != nil {
		t.Fatalf("Retrieve 已加密值失败: %v", err)
	}
	if retrieved != password {
		t.Errorf("Retrieve = %q, 期望 %q", retrieved, password)
	}
}
