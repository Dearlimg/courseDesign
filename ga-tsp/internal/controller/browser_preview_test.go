package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"simple_tuan/internal/dao"
	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

// TestBrowserPreview is an opt-in UI fixture, never used by the application.
func TestBrowserPreview(t *testing.T) {
	if os.Getenv("QIJI_BROWSER_PREVIEW") != "1" {
		t.Skip("browser fixture is opt-in")
	}
	users := &memoryUsers{accounts: map[string]models.Account{}}
	sessions := &memorySessions{items: map[string]memorySession{}}
	addr := os.Getenv("QIJI_PREVIEW_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}
	store := &previewSnapshots{rows: map[uint64]models.Snapshot{}, owners: map[uint64]int64{}}
	server := &http.Server{Addr: addr, Handler: NewApp(logic.NewAuth(users, sessions), false, "../../web", store)}
	t.Cleanup(func() { server.Close() })
	t.Logf("TEST-ONLY browser fixture: http://%s (in-memory accounts and snapshots, no database claim)", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		t.Fatal(err)
	}
}

type previewSnapshots struct {
	mu     sync.Mutex
	rows   map[uint64]models.Snapshot
	owners map[uint64]int64
}

func (s *previewSnapshots) Save(_ context.Context, owner int64, kind string, body json.RawMessage) (models.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uint64(len(s.rows) + 1)
	r := models.Snapshot{ID: id, Kind: kind, Payload: append(json.RawMessage{}, body...), CreatedAt: time.Now()}
	s.rows[id], s.owners[id] = r, owner
	return r, nil
}
func (s *previewSnapshots) List(_ context.Context, owner int64, kind string) ([]models.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := []models.Snapshot{}
	for id, r := range s.rows {
		if s.owners[id] == owner && r.Kind == kind {
			r.Payload = nil
			rows = append(rows, r)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	return rows, nil
}
func (s *previewSnapshots) Get(_ context.Context, owner int64, id uint64) (models.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owners[id] != owner {
		return models.Snapshot{}, dao.ErrNotFound
	}
	return s.rows[id], nil
}
