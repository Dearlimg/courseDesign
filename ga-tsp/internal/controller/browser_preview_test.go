package controller

import (
	"net/http"
	"os"
	"testing"

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
	server := &http.Server{Addr: "127.0.0.1:8081", Handler: NewApp(logic.NewAuth(users, sessions), false, "../../web")}
	t.Cleanup(func() { server.Close() })
	t.Log("TEST-ONLY browser fixture: http://localhost:8081 (in-memory accounts, no database claim)")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		t.Fatal(err)
	}
}
