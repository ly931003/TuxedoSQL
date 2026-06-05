package fileutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigDirName = ".tuxedosql"

// JSONStore 提供基于 JSON 文件的持久化读写。
// 所有文件存储在 ~/.tuxedosql/ 下。
type JSONStore struct {
	dir string
}

// NewJSONStore 创建一个新的 JSONStore。
// 配置目录默认为 ~/.tuxedosql/，首次写入时自动创建。
func NewJSONStore() (*JSONStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}
	return &JSONStore{dir: filepath.Join(homeDir, ConfigDirName)}, nil
}

// ConfigDir 返回 JSONStore 的配置目录路径。
func (s *JSONStore) ConfigDir() string {
	return s.dir
}

// Load 从指定文件反序列化 JSON 到 dest。文件不存在时返回 nil（不报错）。
func (s *JSONStore) Load(filename string, dest interface{}) error {
	path := filepath.Join(s.dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取文件 %s: %w", filename, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析 JSON %s: %w", filename, err)
	}
	return nil
}

// Save 将 data 序列化为格式化的 JSON 并写入文件。目录不存在时自动创建。
func (s *JSONStore) Save(filename string, data interface{}) error {
	path := filepath.Join(s.dir, filename)
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("创建目录 %s: %w", s.dir, err)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON %s: %w", filename, err)
	}

	if err := os.WriteFile(path, bytes, 0600); err != nil {
		return fmt.Errorf("写入文件 %s: %w", filename, err)
	}
	return nil
}
