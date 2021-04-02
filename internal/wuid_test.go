package internal

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/slog"
)

func TestWUID_Next(t *testing.T) {
	const total = 100
	g := NewWUID("default", nil)
	v := atomic.LoadInt64(&g.N)
	for i := 0; i < total; i++ {
		v++
		if id := g.Next(); id != v {
			t.Fatalf("the id is %d, while it should be %d", id, v)
		}
	}
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestWUID_Next_Concurrent(t *testing.T) {
	const total = 10000
	g := NewWUID("default", nil)
	var m sync.Mutex
	var a = make(int64Slice, 0, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func() {
			id := g.Next()
			m.Lock()
			a = append(a, id)
			m.Unlock()
			wg.Done()
		}()
	}

	wg.Wait()
	sort.Sort(a)

	for i := 0; i < total-1; i++ {
		if a[i] == a[i+1] {
			t.Fatalf("duplication detected")
		}
	}
}

func TestWUID_Next_Panic(t *testing.T) {
	const total = 10000
	g := NewWUID("default", nil)
	atomic.StoreInt64(&g.N, PanicValue)

	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func() {
			defer func() {
				_ = recover()
				wg.Done()
			}()

			g.Next()
			t.Fatal("should not be here")
		}()
	}
	wg.Wait()
}

func TestWUID_Next_Renew(t *testing.T) {
	scav := slog.NewScavenger()
	g := NewWUID("default", scav)
	g.Renew = func() error {
		g.Reset(((atomic.LoadInt64(&g.N) >> 36) + 1) << 36)
		return nil
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewIntervalMask) & ^RenewIntervalMask) - 1

	g.Reset((n1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.Reset((n2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>36-n1>>36 != 1 || n3>>36-n2>>36 != 1 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}

	var numInfo int
	scav.Filter(func(level, msg string) bool {
		if level == slog.LevelInfo {
			numInfo++
		}
		return true
	})
	if numInfo != 2 {
		t.Fatalf("there should be 2 renew logs of the info type. actual: %d", numInfo)
	}
}

func TestWUID_Step(t *testing.T) {
	const step = 16
	scav := slog.NewScavenger()
	g := NewWUID("default", scav, WithStep(step, 0))
	g.Reset(17 << 36)
	g.Renew = func() error {
		g.Reset(((atomic.LoadInt64(&g.N) >> 36) + 1) << 36)
		return nil
	}

	for i := int64(1); i < 100; i++ {
		if g.Next()&0x0FFFFFFFFF != step*i {
			t.Fatal("g.Next()&0x0FFFFFFFFF != step*i")
		}
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewIntervalMask) & ^RenewIntervalMask) - 1

	g.Reset(((n1 >> 36 << 36) | kk) & ^(step - 1))
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.Reset(((n2 >> 36 << 36) | kk) & ^(step - 1))
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>36-n1>>36 != 1 || n3>>36-n2>>36 != 1 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}

	var numInfo int
	scav.Filter(func(level, msg string) bool {
		if level == slog.LevelInfo {
			numInfo++
		}
		return true
	})
	if numInfo != 2 {
		t.Fatalf("there should be 2 renew logs of the info type. actual: %d", numInfo)
	}
}

func TestWUID_Floor(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	allSteps := []int64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for loop := 0; loop < 10000; loop++ {
		step := allSteps[r.Intn(len(allSteps))]
		floor := r.Int63n(step)
		scav := slog.NewScavenger()
		g := NewWUID("default", scav, WithStep(step, floor))
		baseValue := r.Int63n(100) << 36
		g.Reset(baseValue)

		for i := int64(1); i < 100; i++ {
			x := g.Next()
			if floor != 0 {
				if reminder := x % floor; reminder != 0 {
					t.Fatal("reminder != 0")
				}
			}
			if x <= baseValue+i*step-step || x > baseValue+i*step {
				t.Fatal("x <= baseValue+i*step-step || x > baseValue+i*step")
			}
		}
	}
}

func TestWUID_Next_Renew_Fail(t *testing.T) {
	scav := slog.NewScavenger()
	g := NewWUID("default", scav)
	g.Renew = func() error {
		return errors.New("foo")
	}

	kk := ((CriticalValue + RenewIntervalMask) & ^RenewIntervalMask) - 1

	g.Reset((1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	g.Next()

	g.Reset((2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	g.Next()

	var numWarn int
	scav = scav.Filter(func(level, msg string) bool {
		if level == slog.LevelWarn {
			numWarn++
		}
		return true
	})
	if numWarn != 2 {
		t.Fatalf("there should be 2 renew logs of the warn type. actual: %d", numWarn)
	}
}

func TestWUID_Next_Renew_Panic(t *testing.T) {
	scav := slog.NewScavenger()
	g := NewWUID("default", scav)
	g.Renew = func() error {
		panic("foo")
	}

	n1 := g.Next()
	kk := ((CriticalValue + RenewIntervalMask) & ^RenewIntervalMask) - 1
	g.Reset((n1 >> 36 << 36) | kk)
	g.Next()

	time.Sleep(time.Millisecond * 200)

	var numWarn int
	scav = scav.Filter(func(level, msg string) bool {
		if level == slog.LevelWarn {
			numWarn++
		}
		return true
	})
	if numWarn != 1 {
		t.Fatalf("there should be 1 renew logs of the warn type. actual: %d", numWarn)
	}
}

func TestWUID_VerifyH28(t *testing.T) {
	g1 := NewWUID("default", nil)
	g1.Reset(0x07FFFFFF << 36)
	if err := g1.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 100, error: %s", err)
	}
	if err := g1.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0")
	}
	if err := g1.VerifyH28(0x08000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0x08000000")
	}
	if err := g1.VerifyH28(0x07FFFFFF); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0x07FFFFFF")
	}

	g2 := NewWUID("default", nil, WithSection(1))
	g2.Reset(0x07FFFFFF << 36)
	if err := g2.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 100, error: %s", err)
	}
	if err := g2.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0")
	}
	if err := g2.VerifyH28(0x01000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0x01000000")
	}
	if err := g2.VerifyH28(0x00FFFFFF); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0x00FFFFFF")
	}
}

func TestWithSection_Panic(t *testing.T) {
	for i := 0; i < 256; i++ {
		func(j int8) {
			defer func() {
				_ = recover()
			}()
			WithSection(j)
			if j >= 8 {
				t.Fatalf("WithSection should only accept values range from 0 to 7. j: %d", j)
			}
		}(int8(i))
	}
}

func TestWithSection_Reset(t *testing.T) {
	for i := 0; i < 28; i++ {
		n := int64(1) << (uint(i) + 36)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if i != 27 {
						t.Fatal(r)
					}
				}
			}()
			for j := int8(1); j < 8; j++ {
				g := NewWUID("default", nil, WithSection(j))
				g.Reset(n)
				v := atomic.LoadInt64(&g.N)
				if v>>60 != int64(j) {
					t.Fatalf("g.Section does not work as expected. g.N: %x, n: %x, i: %d, j: %d", v, n, i, j)
				}
			}
		}()
	}
}

func TestWithRenewCallback(t *testing.T) {
	g := NewWUID("default", nil, WithH28Verifier(func(h28 int64) error {
		if h28 >= 20 {
			return errors.New("bomb")
		}
		return nil
	}))
	if err := g.VerifyH28(10); err != nil {
		t.Fatal("the H28Verifier should not return error")
	}
	if err := g.VerifyH28(20); err == nil || err.Error() != "bomb" {
		t.Fatal("the H28Verifier was not called")
	}
}
