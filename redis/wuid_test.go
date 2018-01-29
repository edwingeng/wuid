package wuid

import (
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/go-redis/redis"
)

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func TestWUID_LoadH24FromRedis(t *testing.T) {
	addr, pass, key := getRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	_, err := client.Del(key).Result()
	if err != nil {
		t.Fatal(err)
	}
	err = client.Close()
	if err != nil {
		t.Fatal(err)
	}

	wuid := NewWUID("default", nil)
	for i := 0; i < 1000; i++ {
		err = wuid.LoadH24FromRedis(getRedisConfig())
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if atomic.LoadUint64(&wuid.w.N) != v {
			t.Fatalf("wuid.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&wuid.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			wuid.Next()
		}
	}
}
