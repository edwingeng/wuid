package wuid

import (
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/go-redis/redis"
)

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func TestWUID_Next(t *testing.T) {
	wuid := NewWUID()
	err := wuid.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}

	v := wuid.n
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
	wuid := NewWUID()
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

	wuid := NewWUID()
	for i := 0; i < 100; i++ {
		err = wuid.LoadH24FromRedis(getRedisConfig())
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if wuid.n != v {
			t.Fatalf("wuid.n is %d, while it should be %d", wuid.n, v)
		}
		for j := 0; j < rand.Intn(10); j++ {
			wuid.Next()
		}
	}
}
