package internal

import (
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWUID_Next(t *testing.T) {
	const total = 100
	g := NewWUID("default", nil)
	v := atomic.LoadUint64(&g.N)
	for i := 0; i < total; i++ {
		v++
		if id := g.Next(); id != v {
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
	g := NewWUID("default", nil)
	var m sync.Mutex
	var a = make(uint64Slice, 0, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			id := g.Next()
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

type simpleLogger struct{}

func (sl simpleLogger) Info(args ...interface{}) {
	log.Println(args...)
}

func (sl simpleLogger) Warn(args ...interface{}) {
	log.Println(args...)
}

func TestWUID_Next_Renew(t *testing.T) {
	g := NewWUID("default", &simpleLogger{})
	g.Renew = func() error {
		atomic.StoreUint64(&g.N, ((atomic.LoadUint64(&g.N)>>40)+1)<<40)
		return nil
	}

	n1 := g.Next()
	kk := ((criticalValue + renewInterval) & ^renewInterval) - 1

	atomic.StoreUint64(&g.N, (n1>>40<<40)|kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	atomic.StoreUint64(&g.N, (n2>>40<<40)|kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n1>>40 == n2>>40 || n2>>40 == n3>>40 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>40, n2>>40, n3>>40)
	}
}

func TestWithSection_Panic(t *testing.T) {
	for i := 0; i < 256; i++ {
		func(j uint8) {
			defer func() {
				recover()
			}()
			WithSection(j)
			if j == 0 || j >= 16 {
				t.Fatalf("WithSection should only accept values range from 1 to 15. j: %d", j)
			}
		}(uint8(i))
	}
}

func TestWUID_Reset(t *testing.T) {
	for i := 0; i < 24; i++ {
		n := uint64(1) << (uint(i) + 40)
		for j := uint8(1); j < 16; j++ {
			g := NewWUID("default", nil, WithSection(j))
			g.Reset(n)
			if j == 0 {
				if atomic.LoadUint64(&g.N) != n {
					t.Fatalf("g.N should not be affected when section == 0")
				}
			} else {
				v := atomic.LoadUint64(&g.N)
				if v>>60 != uint64(j) {
					t.Fatalf("g.Section does not work as expected. g.N: %x, n: %x, i: %d, j: %d", v, n, i, j)
				}
			}
		}
	}
}
