package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"simple_tuan/internal/config"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
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
	users, err := NewMySQLUsers(ctx, c.MySQLAddr, c.MySQLUser, c.MySQLPassword, c.Database, c.CreateDatabase)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := NewRedisSessions(ctx, c.RedisAddr, c.RedisPassword)
	if err != nil {
		users.Close()
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
	defer func() {
		users.db.WithContext(context.Background()).Exec("DELETE FROM qiji_users WHERE id=? AND username=?", user.ID, name)
	}()
	store, err := users.CampusStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(models.OrderBatch{MapVersion: campus.Default().Version, Orders: []models.CampusOrder{}})
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Save(ctx, user.ID, "batch", body)
	if err != nil {
		t.Fatal(err)
	}
	defer users.db.Exec("DELETE FROM tuan_campus_snapshots WHERE id=? AND owner_id=?", record.ID, user.ID)
	loaded, err := store.Get(ctx, user.ID, record.ID)
	if err != nil || string(loaded.Payload) != string(body) {
		t.Fatal("snapshot roundtrip failed")
	}
	if _, err := store.Get(ctx, user.ID+1, record.ID); err != ErrNotFound {
		t.Fatal("snapshot owner isolation failed")
	}
	listed, err := store.List(ctx, user.ID, "batch")
	if err != nil || len(listed) != 1 || listed[0].ID != record.ID {
		t.Fatal("snapshot listing failed")
	}
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
