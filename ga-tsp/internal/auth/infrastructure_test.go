package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gatsp/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// This opt-in test touches only its generated account and exact Redis keys.
func TestInfrastructure(t *testing.T) {
	if os.Getenv("QIJI_INTEGRATION_TEST") != "1" {
		t.Skip("set QIJI_INTEGRATION_TEST=1 to test configured infrastructure")
	}
	if err := config.LoadEnv("../../.env"); err != nil {
		t.Fatal(err)
	}
	c, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	users, sessions, err := Open(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	defer users.Close()
	defer sessions.Close()
	name := fmt.Sprintf("qiji_test_%d", time.Now().UnixNano())
	hash, err := bcrypt.GenerateFromPassword([]byte("integration-only-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := users.Create(ctx, name, hash)
	if err != nil {
		t.Fatal("create test account failed")
	}
	defer users.db.ExecContext(context.Background(), "DELETE FROM qiji_users WHERE id=? AND username=?", user.ID, name)
	account, err := users.Find(ctx, name)
	if err != nil || bcrypt.CompareHashAndPassword(account.PasswordHash, []byte("integration-only-password")) != nil {
		t.Fatal("persisted account verification failed")
	}
	if _, err := users.Create(ctx, name, hash); err != ErrDuplicate {
		t.Fatal("unique constraint failed")
	}
	token := strings.Repeat("a", 32) + fmt.Sprintf("%032x", time.Now().UnixNano())
	if err := sessions.Put(ctx, token, user, time.Second); err != nil {
		t.Fatal("create Redis session failed")
	}
	defer sessions.Delete(context.Background(), token)
	ttl, err := sessions.client.TTL(ctx, sessionKey(token)).Result()
	if err != nil || ttl <= 0 {
		t.Fatal("session TTL missing")
	}
	restored, err := sessions.Get(ctx, token)
	if err != nil || restored.ID != user.ID {
		t.Fatal("session lookup failed")
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := sessions.Get(ctx, token); err != ErrNotFound {
		t.Fatal("session did not expire")
	}
	rateKey := "integration:" + name
	defer sessions.client.Del(context.Background(), "qiji:rate:v1:"+digest(rateKey))
	first, err := sessions.Allow(ctx, rateKey, 1, time.Minute)
	if err != nil || !first {
		t.Fatal("rate limit initial request failed")
	}
	second, err := sessions.Allow(ctx, rateKey, 1, time.Minute)
	if err != nil || second {
		t.Fatal("rate limiting failed")
	}
	t.Log("MySQL account persistence and Redis TTL/rate limiting verified; test records cleaned")
}
