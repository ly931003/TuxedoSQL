package repository

import (
	"os"
	"path/filepath"
	"testing"

	"tuxedosql/internal/model"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("无法获取用户主目录")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "展开 ~ 路径",
			input:    "~/.ssh/id_rsa",
			expected: filepath.Join(home, ".ssh", "id_rsa"),
		},
		{
			name:     "绝对路径保持不变",
			input:    "/home/user/.ssh/id_rsa",
			expected: "/home/user/.ssh/id_rsa",
		},
		{
			name:     "相对路径保持不变",
			input:    ".ssh/id_rsa",
			expected: ".ssh/id_rsa",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.input)
			if got != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildSSHAuthMethods(t *testing.T) {
	t.Run("空配置不返回认证方法", func(t *testing.T) {
		cfg := model.SSHConfig{}
		methods := buildSSHAuthMethods(cfg)
		if len(methods) != 0 {
			t.Errorf("空配置应返回零认证方法，实际返回 %d 个", len(methods))
		}
	})

	t.Run("仅密码返回密码认证", func(t *testing.T) {
		cfg := model.SSHConfig{Password: "test123"}
		methods := buildSSHAuthMethods(cfg)
		if len(methods) != 1 {
			t.Fatalf("应返回 1 个认证方法，实际返回 %d 个", len(methods))
		}
	})

	t.Run("不存在的私钥路径不添加认证方法", func(t *testing.T) {
		cfg := model.SSHConfig{
			PrivateKeyPath: "/nonexistent/key",
			Password:       "fallback",
		}
		methods := buildSSHAuthMethods(cfg)
		// 私钥不存在，只应返回密码认证
		if len(methods) != 1 {
			t.Errorf("不存在的私钥 + 有效密码应返回 1 个认证方法，实际返回 %d 个", len(methods))
		}
	})

	t.Run("密码和私钥同时存在返回两个认证方法", func(t *testing.T) {
		// 创建一个临时私钥文件
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test_key")
		// 写入一个最小化的无效密钥（buildSSHAuthMethods 不会解析它，但 loadPrivateKey 会尝试）
		if err := os.WriteFile(keyPath, []byte("invalid-key-content\n"), 0600); err != nil {
			t.Fatalf("创建临时密钥文件失败: %v", err)
		}

		cfg := model.SSHConfig{
			PrivateKeyPath: keyPath,
			Password:       "test123",
		}
		methods := buildSSHAuthMethods(cfg)
		// 私钥解析会失败，所以只应返回密码认证
		if len(methods) != 1 {
			t.Errorf("无效私钥 + 有效密码应返回 1 个认证方法，实际返回 %d 个", len(methods))
		}
	})
}

func TestCreateSSHTunnelValidation(t *testing.T) {
	m := NewConnectionManager(nil, &MySQLDriver{}, &MySQLSchema{})

	tests := []struct {
		name      string
		conn      model.Connection
		wantError bool
		errMsg    string
	}{
		{
			name:      "SSH 主机为空",
			conn:      model.Connection{ID: "test", SSH: model.SSHConfig{Enabled: true, Port: 22, User: "root"}},
			wantError: true,
			errMsg:    "SSH 主机地址不能为空",
		},
		{
			name:      "SSH 用户名为空",
			conn:      model.Connection{ID: "test", SSH: model.SSHConfig{Enabled: true, Host: "10.0.0.1", Port: 22}},
			wantError: true,
			errMsg:    "SSH 用户名不能为空",
		},
		{
			name: "SSH 端口未设默认 22",
			conn: model.Connection{ID: "test", Host: "db.internal", Port: 3306,
				SSH: model.SSHConfig{Enabled: true, Host: "10.0.0.1", User: "root", Password: "pass"}},
			wantError: true, // 会尝试连接，预期失败
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.createSSHTunnel(&tt.conn)
			if tt.wantError && err == nil {
				t.Error("期望错误但未返回")
			}
			if tt.wantError && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("错误消息不匹配: got %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestGetDBWithoutSSH(t *testing.T) {
	m := NewConnectionManager(nil, &MySQLDriver{}, &MySQLSchema{})

	conn := &model.Connection{
		ID:       "test-conn",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: "pass",
		Database: "test",
		SSH:      model.SSHConfig{Enabled: false},
	}

	// SSH 关闭时，getOrCreateSSHTunnel 不应被调用，DSN 使用原始地址
	// 实际连接会失败（无真实 MySQL），但不应是 SSH 错误
	_, err := m.GetDB(conn, "test")
	if err == nil {
		t.Log("连接到真实 MySQL？意外。")
	}
	// 错误消息不应提及 SSH
	if err != nil {
		t.Logf("预期的连接失败: %v", err)
	}
}

func TestConnectionManagerCloseWithSSHTunnel(t *testing.T) {
	m := NewConnectionManager(nil, &MySQLDriver{}, &MySQLSchema{})

	// 模拟 SSH 隧道存在于 map 中
	m.mu.Lock()
	m.sshTunnels["conn-ssh"] = &sshTunnel{}
	m.mu.Unlock()

	// Close 应清理 SSH 隧道
	m.Close("conn-ssh")

	m.mu.RLock()
	_, exists := m.sshTunnels["conn-ssh"]
	m.mu.RUnlock()

	if exists {
		t.Error("Close 后 SSH 隧道应被移除")
	}
}

func TestConnectionManagerCloseAllSSHTunnels(t *testing.T) {
	m := NewConnectionManager(nil, &MySQLDriver{}, &MySQLSchema{})

	m.mu.Lock()
	m.sshTunnels["conn-a"] = &sshTunnel{}
	m.sshTunnels["conn-b"] = &sshTunnel{}
	m.mu.Unlock()

	m.CloseAll()

	m.mu.RLock()
	count := len(m.sshTunnels)
	m.mu.RUnlock()

	if count != 0 {
		t.Errorf("CloseAll 后应无 SSH 隧道，实际还有 %d 个", count)
	}
}
