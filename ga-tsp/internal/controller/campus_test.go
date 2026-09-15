package controller

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

type snapshotSpy struct{ owner int64 }

func (s *snapshotSpy) Save(_ context.Context, owner int64, kind string, body json.RawMessage) (models.Snapshot, error) {
	s.owner = owner
	return models.Snapshot{ID: 1, Kind: kind, Payload: body}, nil
}
func (s *snapshotSpy) List(_ context.Context, owner int64, _ string) ([]models.Snapshot, error) {
	s.owner = owner
	return []models.Snapshot{}, nil
}
func (s *snapshotSpy) Get(_ context.Context, owner int64, _ uint64) (models.Snapshot, error) {
	s.owner = owner
	return models.Snapshot{}, nil
}

func TestCampusAuthenticationAndStrictInputs(t *testing.T) {
	store := &snapshotSpy{}
	users := &memoryUsers{accounts: map[string]models.Account{}}
	sessions := &memorySessions{items: map[string]memorySession{
		strings.Repeat("a", 64): {user: models.User{ID: 42, Username: "tester"}, expires: time.Now().Add(time.Hour)},
	}}
	h := NewApp(logic.NewAuth(users, sessions), false, "../../web", store)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/campus/history?kind=batch", nil))
	if r.Code != 401 {
		t.Fatal("history has no authentication")
	}
	// Token cookie name is shared with the existing auth handler.
	request := httptest.NewRequest("GET", "/api/campus/history?kind=batch", nil)
	request.Header.Set("Cookie", cookieName+"="+strings.Repeat("a", 64))
	r = httptest.NewRecorder()
	h.ServeHTTP(r, request)
	if r.Code != 200 || store.owner != 42 {
		t.Fatalf("owner must come from session: %d %s", r.Code, r.Body.String())
	}
	for _, body := range []string{`{"count":51}`, `{"count":20,"unknown":true}`} {
		r = httptest.NewRecorder()
		NewMux().ServeHTTP(r, httptest.NewRequest("POST", "/api/campus/batches/generate", strings.NewReader(body)))
		if r.Code != 400 {
			t.Fatal("invalid generation accepted")
		}
	}
}
