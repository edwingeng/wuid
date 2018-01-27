package wuid

import (
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis"
)

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func TestWUID_Next(t *testing.T) {
	wuid := NewWUID("default", nil)
	err := wuid.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}

	v := atomic.LoadUint64(&wuid.n)
	for i := 0; i < 100; i++ {
		v++
		if id := wuid.Next(); id != v {
			t.Fatalf("the id is %d, while it should be %d", id, v)
		}
	}
}

type uint64Slice []uint64

func (p uint64Slice) Len() int           { return len(p) }
func (p uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestWUID_Next_Concurrent(t *testing.T) {
	wuid := NewWUID("default", nil)
	err := wuid.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}

	const total = 100
	var m sync.Mutex
	var a = make(uint64Slice, 0, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			id := wuid.Next()
			m.Lock()
			a = append(a, id)
			m.Unlock()
			wg.Done()
		}(i)
	}

	wg.Wait()
	sort.Sort(a)

	for i := 0; i < total-1; i++ {
		if a[i] == a[i+1] {
			t.Fatalf("duplication detected")
		}
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	wuid := NewWUID("default", log.New(os.Stderr, "", 0))
	err := wuid.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}

	n1 := wuid.Next()
	kk := ((criticalValue + 0x01FFFFFF) & ^uint64(0x01FFFFFF)) - 1

	atomic.StoreUint64(&wuid.n, (n1>>40<<40)|kk)
	wuid.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := wuid.Next()

	atomic.StoreUint64(&wuid.n, (n2>>40<<40)|kk)
	wuid.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := wuid.Next()

	if n1>>40 == n2>>40 || n2>>40 == n3>>40 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>40, n2>>40, n3>>40)
	}
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
	for i := 0; i < 100; i++ {
		err = wuid.LoadH24FromRedis(getRedisConfig())
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if atomic.LoadUint64(&wuid.n) != v {
			t.Fatalf("wuid.n is %d, while it should be %d", atomic.LoadUint64(&wuid.n), v)
		}
		for j := 0; j < rand.Intn(10); j++ {
			wuid.Next()
		}
	}
}
