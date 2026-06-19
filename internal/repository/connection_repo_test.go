package repository

import (
	"strings"
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/fileutil"
)

func newTestConnectionRepository(t *testing.T) (*ConnectionRepository, *fileutil.JSONStore) {
	t.Helper()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	store, err := fileutil.NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}

	return NewConnectionRepository(store), store
}

func loadStoredConnections(t *testing.T, store *fileutil.JSONStore) []model.Connection {
	t.Helper()
	var connections []model.Connection
	if err := store.Load(connectionsFile, &connections); err != nil {
		t.Fatalf("store.Load(%q) error = %v", connectionsFile, err)
	}
	return connections
}

func TestIsLegacyAES(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid prefix with payload", input: "aes256gcm$abc", expected: true},
		{name: "keyring marker", input: "keyring:foo", expected: false},
		{name: "empty", input: "", expected: false},
		{name: "plaintext", input: "hello", expected: false},
		{name: "equal length only", input: "aes256gcm", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLegacyAES(tt.input); got != tt.expected {
				t.Fatalf("isLegacyAES(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConnectionRepoNewConnectionRepository(t *testing.T) {
	repo, _ := newTestConnectionRepository(t)
	if repo == nil {
		t.Fatal("NewConnectionRepository() returned nil")
	}
}

func TestConnectionRepoSaveAndLoadGroupsRoundTrip(t *testing.T) {
	repo, _ := newTestConnectionRepository(t)
	want := []model.ConnectionGroup{{ID: "g1", Name: "Root"}, {ID: "g2", Name: "Child", ParentID: "g1"}, {ID: "g3", Name: "Leaf", ParentID: "g2"}}

	if err := repo.SaveGroups(want); err != nil {
		t.Fatalf("SaveGroups() error = %v", err)
	}

	got, err := repo.LoadGroups()
	if err != nil {
		t.Fatalf("LoadGroups() error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("LoadGroups() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LoadGroups()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestConnectionRepoLoadConnectionsPlaintextPasswordsPreserved(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	want := []model.Connection{{ID: "c1", Name: "one", Password: "plain-1"}, {ID: "c2", Name: "two", Password: "plain-2"}}
	if err := store.Save(connectionsFile, want); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	got, err := repo.LoadConnections()
	if err != nil {
		t.Fatalf("LoadConnections() error = %v", err)
	}
	for i := range want {
		if got[i].Password != want[i].Password {
			t.Fatalf("LoadConnections()[%d].Password = %q, want %q", i, got[i].Password, want[i].Password)
		}
	}
}

func TestConnectionRepoLoadConnectionsEmptyPasswordPreserved(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	if err := store.Save(connectionsFile, []model.Connection{{ID: "c-empty", Name: "empty", Password: ""}}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	got, err := repo.LoadConnections()
	if err != nil {
		t.Fatalf("LoadConnections() error = %v", err)
	}
	if len(got) != 1 || got[0].Password != "" {
		t.Fatalf("LoadConnections() password = %q, want empty string", got[0].Password)
	}
}

func TestConnectionRepoLoadConnectionsKeyringMarkerGraceful(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	connectionID := t.Name()
	repo.DeleteCredential(connectionID)
	if err := store.Save(connectionsFile, []model.Connection{{ID: connectionID, Name: "keyring", Password: "keyring:"}}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	got, err := repo.LoadConnections()
	if err != nil {
		t.Fatalf("LoadConnections() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadConnections() len = %d, want 1", len(got))
	}
	if got[0].Password == "[密钥环不可用]" {
		t.Log("keyring unavailable or entry missing; graceful placeholder returned")
		return
	}
	if got[0].Password == "keyring:" {
		t.Fatal("LoadConnections() left keyring marker unchanged")
	}
}

func TestConnectionRepoLoadConnectionByIDFound(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	want := model.Connection{ID: "found", Name: "found-name", Host: "localhost", Port: 3306, Username: "root", Password: "plain"}
	if err := store.Save(connectionsFile, []model.Connection{want, {ID: "other", Name: "other"}}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	got, err := repo.LoadConnectionByID("found")
	if err != nil {
		t.Fatalf("LoadConnectionByID() error = %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.Password != want.Password {
		t.Fatalf("LoadConnectionByID() = %+v, want %+v", *got, want)
	}
}

func TestConnectionRepoLoadConnectionByIDNotFound(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	if err := store.Save(connectionsFile, []model.Connection{{ID: "exists", Name: "exists"}}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	_, err := repo.LoadConnectionByID("missing")
	if err == nil {
		t.Fatal("LoadConnectionByID() error = nil, want not found error")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("LoadConnectionByID() error = %q, want contains 不存在", err.Error())
	}
}

func TestConnectionRepoSaveConnectionsEmptyPasswordSkipped(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	if err := repo.SaveConnections([]model.Connection{{ID: "c-empty", Name: "empty", Password: ""}}); err != nil {
		t.Fatalf("SaveConnections() error = %v", err)
	}

	got := loadStoredConnections(t, store)
	if len(got) != 1 || got[0].Password != "" {
		t.Fatalf("saved password = %q, want empty string", got[0].Password)
	}
}

func TestConnectionRepoSaveConnectionsLegacyAESSkipped(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	encrypted := "aes256gcm$already-encrypted"
	if err := repo.SaveConnections([]model.Connection{{ID: "c-aes", Name: "aes", Password: encrypted}}); err != nil {
		t.Fatalf("SaveConnections() error = %v", err)
	}

	got := loadStoredConnections(t, store)
	if len(got) != 1 || got[0].Password != encrypted {
		t.Fatalf("saved password = %q, want %q", got[0].Password, encrypted)
	}
}

func TestConnectionRepoDeleteCredentialNoPanic(t *testing.T) {
	repo, _ := newTestConnectionRepository(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DeleteCredential() panicked: %v", r)
		}
	}()
	repo.DeleteCredential("missing-credential")
}

func TestConnectionRepoLoadGroupsEmptyStore(t *testing.T) {
	repo, _ := newTestConnectionRepository(t)
	got, err := repo.LoadGroups()
	if err != nil {
		t.Fatalf("LoadGroups() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadGroups() len = %d, want 0", len(got))
	}
}

func TestConnectionRepoSaveConnectionsPlaintextMigratesToSecureStorage(t *testing.T) {
	repo, store := newTestConnectionRepository(t)
	connection := model.Connection{ID: t.Name(), Name: "plain", Password: "secret-value"}
	t.Cleanup(func() { repo.DeleteCredential(connection.ID) })

	if err := repo.SaveConnections([]model.Connection{connection}); err != nil {
		t.Fatalf("SaveConnections() error = %v", err)
	}

	raw := loadStoredConnections(t, store)
	if len(raw) != 1 {
		t.Fatalf("saved connections len = %d, want 1", len(raw))
	}
	if raw[0].Password == connection.Password {
		t.Fatal("SaveConnections() kept plaintext password")
	}
	if raw[0].Password != "keyring:" && !isLegacyAES(raw[0].Password) {
		t.Fatalf("saved password marker = %q, want keyring: or aes256gcm$...", raw[0].Password)
	}

	loaded, err := repo.LoadConnections()
	if err != nil {
		t.Fatalf("LoadConnections() error = %v", err)
	}
	if loaded[0].Password != connection.Password {
		t.Fatalf("LoadConnections()[0].Password = %q, want %q", loaded[0].Password, connection.Password)
	}
}
