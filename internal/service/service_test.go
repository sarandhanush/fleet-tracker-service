package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestCacheKeyStatus(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"abc", "vehicle:abc:status"},
		{"d9c1", "vehicle:d9c1:status"},
	}
	for _, c := range cases {
		if got := cacheKeyStatus(c.id); got != c.want {
			t.Fatalf("cacheKeyStatus(%s) = %s; want %s", c.id, got, c.want)
		}
	}
}

func TestRandIDUnique(t *testing.T) {
	a := randID()
	b := randID()
	if a == b {
		t.Fatalf("expected unique rand ids, got same: %s", a)
	}
}

func TestRedisSetGetIntegration(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("skip redis integration test - redis not available locally")
	}
	key := "test:fleet:key"
	if err := rdb.Set(ctx, key, `{"ok":true}`, 1*time.Minute).Err(); err != nil {
		t.Fatalf("redis set error: %v", err)
	}
	var s string
	if err := rdb.Get(ctx, key).Scan(&s); err != nil {
		t.Fatalf("redis get error: %v", err)
	}
}
