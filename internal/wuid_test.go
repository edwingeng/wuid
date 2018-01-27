package internal

import (
	"log"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWUID_Next(t *testing.T) {
	const total = 100
	wuid := NewWUID("default", nil)
	v := atomic.LoadUint64(&wuid.N)
	for i := 0; i < total; i++ {
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
	const total = 100
	wuid := NewWUID("default", nil)
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
	wuid.Renew = func() error {
		atomic.StoreUint64(&wuid.N, ((atomic.LoadUint64(&wuid.N)>>40)+1)<<40)
		return nil
	}

	n1 := wuid.Next()
	kk := ((criticalValue + 0x01FFFFFF) & ^uint64(0x01FFFFFF)) - 1

	atomic.StoreUint64(&wuid.N, (n1>>40<<40)|kk)
	wuid.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := wuid.Next()

	atomic.StoreUint64(&wuid.N, (n2>>40<<40)|kk)
	wuid.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := wuid.Next()

	if n1>>40 == n2>>40 || n2>>40 == n3>>40 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>40, n2>>40, n3>>40)
	}
}
