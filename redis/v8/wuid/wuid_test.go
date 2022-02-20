package wuid

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis/v8"
)

var bRedisCluster = flag.Bool("cluster", false, "")

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func getRedisClusterConfig() ([]string, string, string) {
	return []string{"127.0.0.1:6379", "127.0.0.1:6380", "127.0.0.1:6381"}, "", "wuid"
}

func TestWUID_LoadH28FromRedis(t *testing.T) {
	if *bRedisCluster {
		return
	}

	addr, pass, key := getRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer func() {
		_ = client.Close()
	}()

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel1()
	_, err := client.Del(ctx1, key).Result()
	if err != nil {
		t.Fatal(err)
	}
	newClient := func() (redis.UniversalClient, bool, error) {
		return redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
		}), true, nil
	}

	g := NewWUID("default", slog.NewDumbLogger())
	for i := 0; i < 1000; i++ {
		err = g.LoadH28FromRedis(newClient, key)
		if err != nil {
			t.Fatal(err)
		}
		v := (int64(i) + 1) << 36
		if atomic.LoadInt64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH28FromRedis_Error(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", slog.NewDumbLogger())
	if g.LoadH28FromRedis(nil, "") == nil {
		t.Fatal("key is not properly checked")
	}
}

func TestWUID_LoadH28FromRedisCluster(t *testing.T) {
	if !*bRedisCluster {
		return
	}

	addrs, pass, key := getRedisClusterConfig()
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: pass,
	})
	defer func() {
		_ = client.Close()
	}()

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel1()
	_, err := client.Del(ctx1, key).Result()
	if err != nil {
		t.Fatal(err)
	}
	newClient := func() (redis.UniversalClient, bool, error) {
		return client, false, nil
	}

	g := NewWUID("default", slog.NewDumbLogger())
	for i := 0; i < 1000; i++ {
		err = g.LoadH28FromRedis(newClient, key)
		if err != nil {
			t.Fatal(err)
		}
		v := (int64(i) + 1) << 36
		if atomic.LoadInt64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", slog.NewDumbLogger())
	addr, pass, key := getRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer func() {
		_ = client.Close()
	}()
	newClient := func() (redis.UniversalClient, bool, error) {
		return client, false, nil
	}
	err := g.LoadH28FromRedis(newClient, key)
	if err != nil {
		t.Fatal(err)
	}

	n1 := g.Next()
	kk := ((internal.CriticalValue + internal.RenewIntervalMask) & ^internal.RenewIntervalMask) - 1

	g.w.Reset((n1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.w.Reset((n2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>36 == n1>>36 || n3>>36 == n2>>36 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}
}

func TestWithSection(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", slog.NewDumbLogger(), WithSection(7))
	addr, pass, key := getRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer func() {
		_ = client.Close()
	}()
	newClient := func() (redis.UniversalClient, bool, error) {
		return client, false, nil
	}

	err := g.LoadH28FromRedis(newClient, key)
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 7 {
		t.Fatal("WithSection does not work as expected")
	}
}

func Example() {
	newClient := func() (redis.UniversalClient, bool, error) {
		var client redis.UniversalClient
		// ...
		return client, true, nil
	}

	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH28FromRedis(newClient, "wuid")

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
