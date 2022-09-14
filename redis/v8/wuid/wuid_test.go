package wuid

import (
	"errors"
	"flag"
	"fmt"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis/v8"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var redisCluster = flag.Bool("cluster", false, "")

var (
	dumb = slog.NewDumbLogger()
)

var (
	cfg struct {
		addrs    []string
		password string
		key      string
	}
)

func init() {
	cfg.addrs = []string{"127.0.0.1:6379", "127.0.0.1:6380", "127.0.0.1:6381"}
	cfg.key = "v8:wuid"
}

func connect() redis.UniversalClient {
	if *redisCluster {
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.addrs,
			Password: cfg.password,
		})
	} else {
		return redis.NewClient(&redis.Options{
			Addr:     cfg.addrs[0],
			Password: cfg.password,
		})
	}
}

func TestWUID_LoadH28FromRedis(t *testing.T) {
	newClient := func() (redis.UniversalClient, bool, error) {
		return connect(), true, nil
	}
	w := NewWUID("alpha", dumb)
	err := w.LoadH28FromRedis(newClient, cfg.key)
	if err != nil {
		t.Fatal(err)
	}

	initial := atomic.LoadInt64(&w.w.N)
	for i := 1; i < 100; i++ {
		if err := w.RenewNow(); err != nil {
			t.Fatal(err)
		}
		expected := ((initial >> 36) + int64(i)) << 36
		if atomic.LoadInt64(&w.w.N) != expected {
			t.Fatalf("w.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&w.w.N), expected, i)
		}
		n := rand.Intn(10)
		for j := 0; j < n; j++ {
			w.Next()
		}
	}
}

func TestWUID_LoadH28FromRedis_Error(t *testing.T) {
	w := NewWUID("alpha", dumb)
	if w.LoadH28FromRedis(nil, "") == nil {
		t.Fatal("key is not properly checked")
	}

	newErrorClient := func() (redis.UniversalClient, bool, error) {
		return nil, true, errors.New("beta")
	}
	if w.LoadH28FromRedis(newErrorClient, "beta") == nil {
		t.Fatal(`w.LoadH28FromRedis(newErrorClient, "beta") == nil`)
	}
}

func waitUntilNumRenewedReaches(t *testing.T, w *WUID, expected int64) {
	t.Helper()
	startTime := time.Now()
	for time.Since(startTime) < time.Second*3 {
		if atomic.LoadInt64(&w.w.Stats.NumRenewed) == expected {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("timeout")
}

func TestWUID_Next_Renew(t *testing.T) {
	client := connect()
	newClient := func() (redis.UniversalClient, bool, error) {
		return client, false, nil
	}

	w := NewWUID("alpha", slog.NewScavenger())
	err := w.LoadH28FromRedis(newClient, cfg.key)
	if err != nil {
		t.Fatal(err)
	}

	h28 := atomic.LoadInt64(&w.w.N) >> 36
	atomic.StoreInt64(&w.w.N, (h28<<36)|internal.Bye)
	n1a := w.Next()
	if n1a>>36 != h28 {
		t.Fatal(`n1a>>36 != h28`)
	}

	waitUntilNumRenewedReaches(t, w, 1)
	n1b := w.Next()
	if n1b != (h28+1)<<36+1 {
		t.Fatal(`n1b != (h28+1)<<36+1`)
	}

	atomic.StoreInt64(&w.w.N, ((h28+1)<<36)|internal.Bye)
	n2a := w.Next()
	if n2a>>36 != h28+1 {
		t.Fatal(`n2a>>36 != h28+1`)
	}

	waitUntilNumRenewedReaches(t, w, 2)
	n2b := w.Next()
	if n2b != (h28+2)<<36+1 {
		t.Fatal(`n2b != (h28+2)<<36+1`)
	}

	atomic.StoreInt64(&w.w.N, ((h28+2)<<36)|internal.Bye)
	n3a := w.Next()
	if n3a>>36 != h28+2 {
		t.Fatal(`n3a>>36 != h28+2`)
	}

	waitUntilNumRenewedReaches(t, w, 3)
	n3b := w.Next()
	if n3b != (h28+3)<<36+1 {
		t.Fatal(`n3b != (h28+3)<<36+1`)
	}

	atomic.StoreInt64(&w.w.N, ((h28+2)<<36)+internal.Bye+1)
	for i := 0; i < 100; i++ {
		w.Next()
	}
	if atomic.LoadInt64(&w.w.Stats.NumRenewAttempts) != 3 {
		t.Fatal(`atomic.LoadInt64(&w.w.Stats.NumRenewAttempts) != 3`)
	}

	var num int
	sc := w.w.Logger.(*slog.Scavenger)
	sc.Filter(func(level, msg string) bool {
		if level == slog.LevelInfo && strings.Contains(msg, "renew succeeded") {
			num++
		}
		return true
	})
	if num != 3 {
		t.Fatal(`num != 3`)
	}
}

func Example() {
	newClient := func() (redis.UniversalClient, bool, error) {
		var client redis.UniversalClient
		// ...
		return client, true, nil
	}

	// Setup
	w := NewWUID("alpha", nil)
	err := w.LoadH28FromRedis(newClient, "wuid")
	if err != nil {
		panic(err)
	}

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", w.Next())
	}
}
