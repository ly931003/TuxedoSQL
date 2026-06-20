package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewJSONStore(t *testing.T) {
	store, err := NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore 失败: %v", err)
	}
	if store == nil {
		t.Fatal("store 为 nil")
	}
	if store.dir == "" {
		t.Error("dir 为空")
	}
}

func TestJSONStore_SaveAndLoad(t *testing.T) {
	store, err := NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore 失败: %v", err)
	}

	// Use a temp filename to avoid polluting real config
	tmpFile := "test_jsonstore_temp.json"
	defer func() { _ = os.Remove(filepath.Join(store.dir, tmpFile)) }()

	type testData struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	original := testData{Name: "test", Count: 42}

	if err := store.Save(tmpFile, original); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	var loaded testData
	if err := store.Load(tmpFile, &loaded); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	if loaded.Name != original.Name || loaded.Count != original.Count {
		t.Errorf("加载的数据不一致: got %+v, want %+v", loaded, original)
	}
}

func TestJSONStore_LoadNonExistent(t *testing.T) {
	store, err := NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore 失败: %v", err)
	}

	var data []string
	if err := store.Load("non_existent_file_12345.json", &data); err != nil {
		t.Errorf("Load 不存在的文件应返回 nil，但返回了: %v", err)
	}
}

func TestJSONStore_SaveOverwrite(t *testing.T) {
	store, err := NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore 失败: %v", err)
	}

	tmpFile := "test_jsonstore_overwrite.json"
	defer func() { _ = os.Remove(filepath.Join(store.dir, tmpFile)) }()

	if err := store.Save(tmpFile, map[string]int{"a": 1}); err != nil {
		t.Fatalf("首次 Save 失败: %v", err)
	}
	if err := store.Save(tmpFile, map[string]int{"b": 2}); err != nil {
		t.Fatalf("覆盖 Save 失败: %v", err)
	}

	var loaded map[string]int
	if err := store.Load(tmpFile, &loaded); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if loaded["b"] != 2 || loaded["a"] != 0 {
		t.Errorf("覆盖后数据不一致: got %+v", loaded)
	}
}
