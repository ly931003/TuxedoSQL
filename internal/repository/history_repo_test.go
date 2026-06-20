package repository

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/pkg/fileutil"
)

func newTestHistoryRepository(t *testing.T) *HistoryRepository {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	store, err := fileutil.NewJSONStore()
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}

	return NewHistoryRepository(store)
}

func sampleHistoryEntries() []model.QueryHistoryEntry {
	return []model.QueryHistoryEntry{
		{
			ID:           "hist-1",
			ConnectionID: "conn-main",
			Database:     "app_db",
			SQL:          "SELECT * FROM users;",
			Timestamp:    1710000000000,
			Duration:     12,
			RowCount:     10,
			Success:      true,
		},
		{
			ID:           "hist-2",
			ConnectionID: "conn-main",
			Database:     "sales_db",
			SQL:          "UPDATE orders SET total = total + 1 WHERE id = 7;",
			Timestamp:    1710000001000,
			Duration:     18,
			RowCount:     1,
			Success:      true,
		},
		{
			ID:           "hist-3",
			ConnectionID: "conn-analytics",
			Database:     "audit_db",
			SQL:          "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 100;",
			Timestamp:    1710000002000,
			Duration:     33,
			RowCount:     100,
			Success:      false,
		},
	}
}

func requireHistoryEntriesEqual(t *testing.T, got, want []model.QueryHistoryEntry) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("history mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestHistoryRepo(t *testing.T) {
	t.Run("TestNewHistoryRepository", func(t *testing.T) {
		if repo := newTestHistoryRepository(t); repo == nil {
			t.Fatal("NewHistoryRepository() returned nil")
		}
	})

	t.Run("TestHistoryRepoLoadEmptyStore", func(t *testing.T) {
		repo := newTestHistoryRepository(t)

		entries, err := repo.LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}
		if entries == nil {
			t.Fatal("LoadHistory() returned nil, want empty slice")
		}
		if len(entries) != 0 {
			t.Fatalf("LoadHistory() len = %d, want 0", len(entries))
		}
	})

	t.Run("TestHistoryRepoSaveLoadRoundTrip", func(t *testing.T) {
		repo := newTestHistoryRepository(t)
		want := sampleHistoryEntries()

		if err := repo.SaveHistory(want); err != nil {
			t.Fatalf("SaveHistory() error = %v", err)
		}

		got, err := repo.LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}

		requireHistoryEntriesEqual(t, got, want)
	})

	t.Run("TestHistoryRepoSaveOverwrite", func(t *testing.T) {
		repo := newTestHistoryRepository(t)
		first := sampleHistoryEntries()
		second := first[:1]

		if err := repo.SaveHistory(first); err != nil {
			t.Fatalf("first SaveHistory() error = %v", err)
		}
		if err := repo.SaveHistory(second); err != nil {
			t.Fatalf("second SaveHistory() error = %v", err)
		}

		got, err := repo.LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}

		requireHistoryEntriesEqual(t, got, second)
	})

	t.Run("TestHistoryRepoSaveNilSlice", func(t *testing.T) {
		repo := newTestHistoryRepository(t)

		if err := repo.SaveHistory(nil); err != nil {
			t.Fatalf("SaveHistory(nil) error = %v", err)
		}

		got, err := repo.LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}
		if got == nil {
			t.Fatal("LoadHistory() returned nil, want empty slice")
		}
		if len(got) != 0 {
			t.Fatalf("LoadHistory() len = %d, want 0", len(got))
		}
	})

	t.Run("TestHistoryRepoConcurrentReadWhileWrite", func(t *testing.T) {
		repo := newTestHistoryRepository(t)

		if err := repo.SaveHistory(sampleHistoryEntries()); err != nil {
			t.Fatalf("initial SaveHistory() error = %v", err)
		}

		start := make(chan struct{})
		errCh := make(chan error, 11)
		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() {
				<-start
				for range 50 {
					entries, err := repo.LoadHistory()
					if err != nil {
						errCh <- err
						return
					}
					if len(entries) == 0 {
						errCh <- fmt.Errorf("LoadHistory() returned no history entries")
						return
					}
				}
			})
		}

		wg.Go(func() {
			<-start
			for i := range 25 {
				writeEntries := []model.QueryHistoryEntry{{
					ID:           fmt.Sprintf("hist-%d", i),
					ConnectionID: "conn-concurrent",
					Database:     "race_db",
					SQL:          fmt.Sprintf("SELECT %d;", i),
					Timestamp:    int64(i),
					Duration:     int64(i + 1),
					RowCount:     int64(i + 2),
					Success:      i%2 == 0,
				}}

				if err := repo.SaveHistory(writeEntries); err != nil {
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
	})
}
