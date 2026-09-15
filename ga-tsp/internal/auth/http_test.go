package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type memoryUsers struct {
	accounts map[string]Account
	mu       sync.Mutex
}

func (m *memoryUsers) Create(_ context.Context, name string, hash []byte) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.accounts[name]; ok {
		return User{}, ErrDuplicate
	}
	u := User{ID: int64(len(m.accounts) + 1), Username: name}
	m.accounts[name] = Account{User: u, PasswordHash: append([]byte{}, hash...)}
	return u, nil
}
func (m *memoryUsers) Find(_ context.Context, name string) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[name]
	if !ok {
		return a, ErrNotFound
	}
	return a, nil
}

type memorySession struct {
	user    User
	expires time.Time
}
type memorySessions struct {
	mu      sync.Mutex
	items   map[string]memorySession
	fail    bool
	blocked bool
}

func (m *memorySessions) Put(_ context.Context, token string, u User, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("offline")
	}
	m.items[token] = memorySession{user: u, expires: time.Now().Add(ttl)}
	return nil
}
func (m *memorySessions) Get(_ context.Context, token string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return User{}, errors.New("offline")
	}
	s, ok := m.items[token]
	if !ok || s.expires.Before(time.Now()) {
		return User{}, ErrNotFound
	}
	return s.user, nil
}
func (m *memorySessions) Delete(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("offline")
	}
	delete(m.items, token)
	return nil
}
func (m *memorySessions) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return false, errors.New("offline")
	}
	return !m.blocked, nil
}
func request(h http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAuthenticationLifecycle(t *testing.T) {
	users := &memoryUsers{accounts: map[string]Account{}}
	sessions := &memorySessions{items: map[string]memorySession{}}
	s := New(users, sessions, false)
	h := s.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	body := `{"username":"Test_User","password":"test-password-123"}`
	if w := request(h, "GET", "/api/dispatch/anything", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/", "", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := request(h, "POST", "/api/auth/register", body, nil); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	if bcrypt.CompareHashAndPassword(users.accounts["test_user"].PasswordHash, []byte("test-password-123")) != nil {
		t.Fatal("password hash incorrect")
	}
	if w := request(h, "POST", "/api/auth/register", body, nil); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := request(h, "POST", "/api/auth/login", `{"username":"test_user","password":"wrong-password"}`, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := request(h, "POST", "/api/auth/login", body, nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || len(cookie.Value) != 64 || cookie.MaxAge != 86400 {
		t.Fatal("cookie flags incorrect")
	}
	if strings.Contains(w.Body.String(), cookie.Value) || strings.Contains(w.Body.String(), "password") {
		t.Fatal("credential leaked")
	}
	if w := request(h, "GET", "/api/auth/me", "", cookie); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/api/dispatch/anything", "", cookie); w.Code != 204 {
		t.Fatal(w.Code)
	}
	rotated := request(h, "POST", "/api/auth/login", body, cookie).Result().Cookies()[0]
	if cookie.Value == rotated.Value {
		t.Fatal("token not rotated")
	}
	if w := request(h, "GET", "/api/auth/me", "", cookie); w.Code != 401 {
		t.Fatal("old token accepted")
	}
	if w := request(h, "POST", "/api/auth/logout", "", rotated); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/api/auth/me", "", rotated); w.Code != 401 {
		t.Fatal("logout failed")
	}
	expired := memorySession{user: User{ID: 1}, expires: time.Now().Add(-time.Second)}
	sessions.items[cookie.Value] = expired
	if w := request(h, "GET", "/api/auth/me", "", cookie); w.Code != 401 {
		t.Fatal("expired accepted")
	}
	sessions.fail = true
	if w := request(h, "GET", "/api/dispatch/anything", "", cookie); w.Code != 503 {
		t.Fatal("redis failure allowed access")
	}
}
func TestAuthValidationAndOrigin(t *testing.T) {
	sessions := &memorySessions{items: map[string]memorySession{}}
	h := New(&memoryUsers{accounts: map[string]Account{}}, sessions, true).Handler(http.NotFoundHandler())
	for _, body := range []string{`{"username":"a","password":"12345678"}`, `{"username":"valid","password":"short"}`, `{"username":"bad' OR 1=1","password":"12345678"}`, `{}`} {
		if w := request(h, "POST", "/api/auth/register", body, nil); w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("{}"))
	r.Header.Set("Origin", "https://other.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin allowed")
	}
	sessions.blocked = true
	if w := request(h, "POST", "/api/auth/login", `{}`, nil); w.Code != 429 {
		t.Fatal(w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
}
