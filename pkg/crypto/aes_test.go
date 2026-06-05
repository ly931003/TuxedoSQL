package crypto

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	keyHex, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("解码密钥失败: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"普通密码", "myPassword123"},
		{"空字符串", ""},
		{"中文密码", "我的密码@#$%"},
		{"特殊字符", "p@$$w0rd!~`[]{}|\\;:'\",.<>?/"},
		{"长密码", strings.Repeat("x", 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("加密失败: %v", err)
			}
			if ciphertext == "" {
				t.Error("密文不应为空")
			}

			plaintext, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("解密失败: %v", err)
			}
			if plaintext != tt.plaintext {
				t.Errorf("解密结果 = %q, 期望 %q", plaintext, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_DeterministicOutput(t *testing.T) {
	keyHex, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("解码密钥失败: %v", err)
	}

	// 两次加密同一明文应产生不同密文（随机 nonce）
	c1, _ := Encrypt("password", key)
	c2, _ := Encrypt("password", key)
	if c1 == c2 {
		t.Error("同一明文两次加密应产生不同密文（随机 nonce）")
	}
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	_, err := Encrypt("test", make([]byte, 16))
	if err == nil {
		t.Error("16 字节密钥应报错")
	}

	_, err = Decrypt("test", make([]byte, 16))
	if err == nil {
		t.Error("16 字节密钥解密应报错")
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	keyHex, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("解码密钥失败: %v", err)
	}

	_, err = Decrypt("not-hex-@@@", key)
	if err == nil {
		t.Error("非法 hex 应报错")
	}

	// 太短的密文
	shortHex := hex.EncodeToString([]byte("short"))
	_, err = Decrypt(shortHex, key)
	if err == nil {
		t.Error("太短的密文应报错")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1Hex, _ := GenerateKey()
	key2Hex, _ := GenerateKey()
	key1, _ := hex.DecodeString(key1Hex)
	key2, _ := hex.DecodeString(key2Hex)

	ciphertext, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("使用错误密钥解密应报错")
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"aes256gcm$xxx", true},
		{"aes256gcm$", true},
		{"aes256gcm", false},
		{"plaintext", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsEncrypted(tt.input); got != tt.expected {
				t.Errorf("IsEncrypted(%q) = %v, 期望 %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateKey(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	k2, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}

	if k1 == k2 {
		t.Error("两次生成的密钥不应相同")
	}

	// 验证 hex 长度（32 字节 → 64 hex 字符）
	if len(k1) != 64 {
		t.Errorf("密钥 hex 长度 = %d, 期望 64", len(k1))
	}
}
