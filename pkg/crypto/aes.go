// Package crypto 提供 AES-256-GCM 加密/解密工具，用于保护存储在本地配置中的敏感数据。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

const nonceSize = 12 // GCM 推荐的 nonce 长度

// Encrypt 使用 AES-256-GCM 加密明文，返回 hex 编码的密文（格式：nonce+ciphertext）。
// key 必须是 32 字节（AES-256）。
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("密钥长度必须为32字节，当前 %d 字节", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机 nonce 失败: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt 解密 hex 编码的密文，返回明文。
// 密文格式为 nonce+ciphertext，由 Encrypt 生成。
func Decrypt(hexCiphertext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("密钥长度必须为32字节，当前 %d 字节", len(key))
	}

	ciphertext, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("hex 解码失败: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("密文长度不足")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// GenerateKey 生成一个新的 AES-256 随机密钥（32 字节），hex 编码返回。
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("生成密钥失败: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// EncryptedPrefix 是已加密密文的前缀标记，用于区分明文和密文。
const EncryptedPrefix = "aes256gcm$"

// IsEncrypted 判断字符串是否已加密（以 EncryptedPrefix 开头）。
func IsEncrypted(s string) bool {
	return len(s) >= len(EncryptedPrefix) && s[:len(EncryptedPrefix)] == EncryptedPrefix
}
