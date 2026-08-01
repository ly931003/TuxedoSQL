package repository

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"tuxedosql/internal/model"
)

// sshTunnel 表示一个活跃的 SSH 端口转发隧道。
// 通过 localhost:localPort 将 TCP 流量转发到 remoteHost:remotePort。
type sshTunnel struct {
	client    *ssh.Client
	listener  net.Listener
	localPort int
	cancel    context.CancelFunc // 停止转发循环
	done      chan struct{}      // 转发循环已退出
}

// getOrCreateSSHTunnel 获取或创建指定连接的 SSH 隧道。
// 返回本地监听端口，调用方通过 127.0.0.1:localPort 连接数据库。
func (m *ConnectionManager) getOrCreateSSHTunnel(conn *model.Connection) (int, error) {
	if !conn.SSH.Enabled {
		return 0, fmt.Errorf("SSH 隧道未启用")
	}

	// 快速路径：读锁检查是否已存在
	m.mu.RLock()
	if t, ok := m.sshTunnels[conn.ID]; ok {
		m.mu.RUnlock()
		return t.localPort, nil
	}
	m.mu.RUnlock()

	// 慢速路径：创建新隧道
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if t, ok := m.sshTunnels[conn.ID]; ok {
		return t.localPort, nil
	}

	tunnel, err := m.createSSHTunnel(conn)
	if err != nil {
		return 0, err
	}

	m.sshTunnels[conn.ID] = tunnel
	return tunnel.localPort, nil
}

// createSSHTunnel 建立到 SSH 服务器的连接，并启动本地端口转发。
func (m *ConnectionManager) createSSHTunnel(conn *model.Connection) (*sshTunnel, error) {
	cfg := conn.SSH

	// 验证必填字段
	if cfg.Host == "" {
		return nil, fmt.Errorf("SSH 主机地址不能为空")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.User == "" {
		return nil, fmt.Errorf("SSH 用户名不能为空")
	}

	// 构建 SSH 客户端配置
	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("加载 known_hosts 失败: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            buildSSHAuthMethods(cfg),
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	// 连接到 SSH 服务器
	sshAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败 (%s): %w", sshAddr, err)
	}

	// 在本地监听随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("创建本地监听失败: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port

	// 启动转发循环
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	targetAddr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	go func() {
		defer close(done)
		m.forwardLoop(ctx, listener, sshClient, targetAddr)
	}()

	return &sshTunnel{
		client:    sshClient,
		listener:  listener,
		localPort: localPort,
		cancel:    cancel,
		done:      done,
	}, nil
}

// forwardLoop 接受本地连接并通过 SSH 隧道转发到目标地址。
func (m *ConnectionManager) forwardLoop(ctx context.Context, listener net.Listener, sshClient *ssh.Client, targetAddr string) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 设置 Accept 超时，以便检查 ctx 取消
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			_ = tcpListener.SetDeadline(time.Now().Add(500 * time.Millisecond))
		}

		local, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue // 临时错误，重试
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			m.handleForward(ctx, local, sshClient, targetAddr)
		}()
	}
}

// handleForward 处理单次端口转发：local <-> SSH tunnel <-> remote。
func (m *ConnectionManager) handleForward(ctx context.Context, local net.Conn, sshClient *ssh.Client, targetAddr string) {
	defer func() { _ = local.Close() }()

	remote, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer func() { _ = remote.Close() }()

	// 双向拷贝
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(remote, local)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(local, remote)
		done <- struct{}{}
	}()

	// 等待任一方向完成，或 context 取消
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// closeSSHTunnel 关闭并移除指定连接的 SSH 隧道。
func (m *ConnectionManager) closeSSHTunnel(connectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel, ok := m.sshTunnels[connectionID]
	if !ok {
		return
	}

	if tunnel.cancel != nil {
		tunnel.cancel()
	}
	if tunnel.listener != nil {
		_ = tunnel.listener.Close()
	}
	select {
	case <-tunnel.done:
	case <-time.After(2 * time.Second):
	}
	if tunnel.client != nil {
		_ = tunnel.client.Close()
	}
	delete(m.sshTunnels, connectionID)
}

// closeAllSSHTunnel 关闭所有 SSH 隧道。
func (m *ConnectionManager) closeAllSSHTunnel() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, tunnel := range m.sshTunnels {
		if tunnel.cancel != nil {
			tunnel.cancel()
		}
		if tunnel.listener != nil {
			_ = tunnel.listener.Close()
		}
		select {
		case <-tunnel.done:
		case <-time.After(2 * time.Second):
		}
		if tunnel.client != nil {
			_ = tunnel.client.Close()
		}
		delete(m.sshTunnels, id)
	}
}

// buildSSHAuthMethods 根据配置构建 SSH 认证方法列表。
// 私钥优先，密码回退。
func buildSSHAuthMethods(cfg model.SSHConfig) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// 1. 私钥认证（优先）
	if cfg.PrivateKeyPath != "" {
		if method := loadPrivateKey(cfg.PrivateKeyPath, cfg.PrivateKeyPass); method != nil {
			methods = append(methods, method)
		}
	}

	// 2. 密码认证
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	return methods
}

// loadPrivateKey 从文件加载私钥，支持加密私钥（通过 PrivateKeyPass 解密）。
func loadPrivateKey(path, passphrase string) ssh.AuthMethod {
	expandedPath := expandPath(path)
	keyBytes, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil
	}

	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
	}
	if err != nil {
		return nil
	}

	return ssh.PublicKeys(signer)
}

// expandPath 展开路径中的 ~ 为用户主目录。
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

// buildHostKeyCallback 构建 SSH 主机密钥验证回调。
// 优先从 ~/.ssh/known_hosts 加载，回退到 /etc/ssh/ssh_known_hosts。
// 如果两者都不可用，返回错误拒绝连接。
func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userKnownHosts := filepath.Join(homeDir, ".ssh", "known_hosts")
		if _, statErr := os.Stat(userKnownHosts); statErr == nil {
			return knownhosts.New(userKnownHosts)
		}
	}

	systemKnownHosts := "/etc/ssh/ssh_known_hosts"
	if _, statErr := os.Stat(systemKnownHosts); statErr == nil {
		return knownhosts.New(systemKnownHosts)
	}

	return nil, fmt.Errorf("未找到 known_hosts 文件（~/.ssh/known_hosts 或 %s），无法验证 SSH 主机密钥。"+
		"请先通过 ssh 命令连接目标服务器以将其主机密钥添加到 known_hosts", systemKnownHosts)
}
