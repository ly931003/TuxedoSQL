package repository

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/fileutil"
)

func newTestTabRepository(t *testing.T) *TabRepository {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	store, err := fileutil.NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}
	return NewTabRepository(store)
}

func sampleTabs() []model.TabState {
	return []model.TabState{{ID: "tab-1", Title: "Users", ConnectionID: "conn-main", Database: "app_db", SQL: "SELECT * FROM users;"}, {ID: "tab-2", Title: "Orders", ConnectionID: "conn-main", Database: "sales_db", SQL: "SELECT id, total FROM orders WHERE total > 100;"}, {ID: "tab-3", Title: "Audit", ConnectionID: "conn-analytics", Database: "audit_db", SQL: "SELECT * FROM audit_logs ORDER BY created_at DESC;"}}
}

func requireTabsEqual(t *testing.T, got, want []model.TabState) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tabs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestTabRepoNewTabRepository(t *testing.T) {
	if repo := newTestTabRepository(t); repo == nil {
		t.Fatal("NewTabRepository() returned nil")
	}
}

func TestTabRepoLoadTabsFreshStore(t *testing.T) {
	repo := newTestTabRepository(t)
	tabs, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	if len(tabs) != 0 {
		t.Fatalf("LoadTabs() len = %d, want 0", len(tabs))
	}
}

func TestTabRepoSaveTabsLoadTabsRoundTrip(t *testing.T) {
	repo := newTestTabRepository(t)
	want := sampleTabs()
	if err := repo.SaveTabs(want); err != nil {
		t.Fatalf("SaveTabs() error = %v", err)
	}
	got, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	requireTabsEqual(t, got, want)
}

func TestTabRepoSaveTabsEmptySlice(t *testing.T) {
	repo := newTestTabRepository(t)
	if err := repo.SaveTabs([]model.TabState{}); err != nil {
		t.Fatalf("SaveTabs() error = %v", err)
	}
	got, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadTabs() len = %d, want 0", len(got))
	}
}

func TestTabRepoSaveTabsSingleTab(t *testing.T) {
	repo := newTestTabRepository(t)
	want := []model.TabState{{ID: "tab-single", Title: "Scratch Query", ConnectionID: "conn-dev", Database: "sandbox", SQL: "SELECT NOW();"}}
	if err := repo.SaveTabs(want); err != nil {
		t.Fatalf("SaveTabs() error = %v", err)
	}
	got, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	requireTabsEqual(t, got, want)
}

func TestTabRepoSaveTabsOverwriteExistingTabs(t *testing.T) {
	repo := newTestTabRepository(t)
	first := sampleTabs()
	second := first[:1]
	if err := repo.SaveTabs(first); err != nil {
		t.Fatalf("first SaveTabs() error = %v", err)
	}
	if err := repo.SaveTabs(second); err != nil {
		t.Fatalf("second SaveTabs() error = %v", err)
	}

	got, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	requireTabsEqual(t, got, second)
}

func TestTabRepoSaveTabsNilSlice(t *testing.T) {
	repo := newTestTabRepository(t)
	if err := repo.SaveTabs(nil); err != nil {
		t.Fatalf("SaveTabs(nil) error = %v", err)
	}
	got, err := repo.LoadTabs()
	if err != nil {
		t.Fatalf("LoadTabs() error = %v", err)
	}
	if got != nil {
		t.Fatalf("LoadTabs() = %#v, want nil slice after saving nil", got)
	}
}

func TestTabRepoConcurrentReadWhileWrite(t *testing.T) {
	repo := newTestTabRepository(t)
	if err := repo.SaveTabs(sampleTabs()); err != nil {
		t.Fatalf("initial SaveTabs() error = %v", err)
	}
	start := make(chan struct{})
	errCh := make(chan error, 11)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			<-start
			for range 50 {
				tabs, err := repo.LoadTabs()
				if err != nil {
					errCh <- err
					return
				}
				if len(tabs) == 0 {
					errCh <- fmt.Errorf("LoadTabs() returned no tabs")
					return
				}
			}
		})
	}
	wg.Go(func() {
		<-start
		for i := range 25 {
			writeTabs := []model.TabState{{ID: fmt.Sprintf("tab-%d", i), Title: fmt.Sprintf("Query %d", i), ConnectionID: "conn-concurrent", Database: "race_db", SQL: fmt.Sprintf("SELECT %d;", i)}}
			if err := repo.SaveTabs(writeTabs); err != nil {
				errCh <- err
				return
			}
		}
	})
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
