package internal

import (
	"errors"
	"fmt"
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
	const total = 1000
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

type simpleLogger struct {
	numInfo int
	numWarn int
}

func (this *simpleLogger) Info(args ...interface{}) {
	this.numInfo++
	if !testing.Verbose() {
		return
	}
	str := "INFO\t"
	str += fmt.Sprint(args...)
	log.Println(str)
}

func (this *simpleLogger) Warn(args ...interface{}) {
	this.numWarn++
	if !testing.Verbose() {
		return
	}
	str := "WARN\t"
	str += fmt.Sprint(args...)
	log.Println(str)
}

func TestWUID_Next_Panic(t *testing.T) {
	defer func() {
		_ = recover()
	}()

	g := NewWUID("default", nil)
	atomic.StoreUint64(&g.N, PanicValue)
	g.Next()

	t.Fatal("should not be here")
}

func TestWUID_Next_Renew(t *testing.T) {
	logger := &simpleLogger{}
	g := NewWUID("default", logger)
	g.Renew = func() error {
		g.Reset(((atomic.LoadUint64(&g.N) >> 36) + 1) << 36)
		return nil
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewInterval) & ^RenewInterval) - 1

	g.Reset((n1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.Reset((n2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>36 == n1>>36 || n3>>36 == n2>>36 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}
	if logger.numInfo != 2 {
		t.Fatalf("there should be 2 renew logs of the info type. actual: %d", logger.numInfo)
	}
}

func TestWUID_Next_Renew_Fail(t *testing.T) {
	logger := &simpleLogger{}
	g := NewWUID("default", logger)
	g.Renew = func() error {
		return errors.New("foo")
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewInterval) & ^RenewInterval) - 1

	g.Reset((n1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.Reset((n2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	g.Next()

	if logger.numWarn != 2 {
		t.Fatalf("there should be 2 renew logs of the warn type. actual: %d", logger.numWarn)
	}
}

func TestWUID_Next_Renew_Panic(t *testing.T) {
	g := NewWUID("default", &simpleLogger{})
	g.Renew = func() error {
		panic("foo")
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewInterval) & ^RenewInterval) - 1
	g.Reset((n1 >> 36 << 36) | kk)
	g.Next()

	time.Sleep(time.Millisecond * 200)
}

func TestWUID_VerifyH28(t *testing.T) {
	g1 := NewWUID("default", nil)
	if err := g1.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 100, error: %s", err)
	}
	if err := g1.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0")
	}
	if err := g1.VerifyH28(0x10000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0x10000000")
	}

	g2 := NewWUID("default", nil, WithSection(1))
	if err := g2.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 100, error: %s", err)
	}
	if err := g2.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0")
	}
	if err := g2.VerifyH28(0x1000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0x1000000")
	}
}

func TestWithSection_Panic(t *testing.T) {
	for i := 0; i < 256; i++ {
		func(j uint8) {
			defer func() {
				_ = recover()
			}()
			WithSection(j)
			if j == 0 || j >= 16 {
				t.Fatalf("WithSection should only accept values range from 1 to 15. j: %d", j)
			}
		}(uint8(i))
	}
}

func TestWithSection_Reset(t *testing.T) {
	for i := 0; i < 28; i++ {
		n := uint64(1) << (uint(i) + 36)
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

func TestWithRenewCallback(t *testing.T) {
	g := NewWUID("default", nil, WithH28Verifier(func(h28 uint64) error {
		if h28 >= 10 {
			return errors.New("bomb")
		}
		return nil
	}))
	if err := g.VerifyH28(10); err == nil || err.Error() != "bomb" {
		t.Fatal("the H28Verifier was not called")
	}
}
