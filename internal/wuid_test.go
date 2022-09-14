package internal

import (
	"errors"
	"github.com/edwingeng/slog"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func (w *WUID) Scavenger() *slog.Scavenger {
	return w.Logger.(*slog.Scavenger)
}

func TestWUID_Next(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := NewWUID("alpha", nil)
		w.Reset(int64(i+1) << 36)
		v := atomic.LoadInt64(&w.N)
		for j := 0; j < 100; j++ {
			v++
			if id := w.Next(); id != v {
				t.Fatalf("the id is %d, while it should be %d", id, v)
			}
		}
	}
}

func TestWUID_Next_Concurrent(t *testing.T) {
	w := NewWUID("alpha", nil)
	var mu sync.Mutex
	const N1 = 100
	const N2 = 100
	a := make([]int64, 0, N1*N2)

	var wg sync.WaitGroup
	for i := 0; i < N1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < N2; j++ {
				id := w.Next()
				mu.Lock()
				a = append(a, id)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	sort.Slice(a, func(i, j int) bool {
		return a[i] < a[j]
	})

	for i := 0; i < N1*N2-1; i++ {
		if a[i] == a[i+1] {
			t.Fatalf("duplication detected")
		}
	}
}

func TestWUID_Next_Panic(t *testing.T) {
	const total = 100
	w := NewWUID("alpha", nil)
	atomic.StoreInt64(&w.N, PanicValue)

	ch := make(chan int64, total)
	for i := 0; i < total; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					ch <- 0
				}
			}()

			ch <- w.Next()
		}()
	}

	for i := 0; i < total; i++ {
		v := <-ch
		if v != 0 {
			t.Fatal("something is wrong with Next()")
		}
	}
}

func waitUntilNumRenewAttemptsReaches(t *testing.T, w *WUID, expected int64) {
	t.Helper()
	startTime := time.Now()
	for time.Since(startTime) < time.Second {
		if atomic.LoadInt64(&w.Stats.NumRenewAttempts) == expected {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("timeout")
}

func waitUntilNumRenewedReaches(t *testing.T, w *WUID, expected int64) {
	t.Helper()
	startTime := time.Now()
	for time.Since(startTime) < time.Second {
		if atomic.LoadInt64(&w.Stats.NumRenewed) == expected {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("timeout")
}

func TestWUID_Renew(t *testing.T) {
	w := NewWUID("alpha", slog.NewScavenger())
	w.Renew = func() error {
		w.Reset(((atomic.LoadInt64(&w.N) >> 36) + 1) << 36)
		return nil
	}

	w.Reset(Bye)
	n1a := w.Next()
	if n1a>>36 != 0 {
		t.Fatal(`n1a>>36 != 0`)
	}

	waitUntilNumRenewedReaches(t, w, 1)
	n1b := w.Next()
	if n1b != 1<<36+1 {
		t.Fatal(`n1b != 1<<36+1`)
	}

	w.Reset(1<<36 | Bye)
	n2a := w.Next()
	if n2a>>36 != 1 {
		t.Fatal(`n2a>>36 != 1`)
	}

	waitUntilNumRenewedReaches(t, w, 2)
	n2b := w.Next()
	if n2b != 2<<36+1 {
		t.Fatal(`n2b != 2<<36+1`)
	}

	w.Reset(2<<36 | Bye + RenewIntervalMask + 1)
	n3a := w.Next()
	if n3a>>36 != 2 {
		t.Fatal(`n3a>>36 != 2`)
	}

	waitUntilNumRenewedReaches(t, w, 3)
	n3b := w.Next()
	if n3b != 3<<36+1 {
		t.Fatal(`n3b != 3<<36+1`)
	}

	w.Reset(Bye + 1)
	for i := 0; i < 100; i++ {
		w.Next()
	}
	if atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 3 {
		t.Fatal(`atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 3`)
	}

	var num int
	w.Scavenger().Filter(func(level, msg string) bool {
		if level == slog.LevelInfo && strings.Contains(msg, "renew succeeded") {
			num++
		}
		return true
	})
	if num != 3 {
		t.Fatal(`num != 3`)
	}
}

func TestWUID_Renew_Error(t *testing.T) {
	w := NewWUID("alpha", slog.NewScavenger())
	w.Renew = func() error {
		return errors.New("foo")
	}

	w.Reset((1 >> 36 << 36) | Bye)
	w.Next()
	waitUntilNumRenewAttemptsReaches(t, w, 1)
	w.Next()

	w.Reset((2 >> 36 << 36) | Bye)
	w.Next()
	waitUntilNumRenewAttemptsReaches(t, w, 2)

	for i := 0; i < 100; i++ {
		w.Next()
	}
	if atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 2 {
		t.Fatal(`atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 2`)
	}
	if atomic.LoadInt64(&w.Stats.NumRenewed) != 0 {
		t.Fatal(`atomic.LoadInt64(&w.Stats.NumRenewed) != 0`)
	}

	var num int
	w.Scavenger().Filter(func(level, msg string) bool {
		if level == slog.LevelWarn && strings.Contains(msg, "renew failed") && strings.Contains(msg, "foo") {
			num++
		}
		return true
	})
	if num != 2 {
		t.Fatal(`num != 2`)
	}
}

func TestWUID_Renew_Panic(t *testing.T) {
	w := NewWUID("alpha", slog.NewScavenger())
	w.Renew = func() error {
		panic("foo")
	}

	w.Reset((1 >> 36 << 36) | Bye)
	w.Next()
	waitUntilNumRenewAttemptsReaches(t, w, 1)
	w.Next()

	w.Reset((2 >> 36 << 36) | Bye)
	w.Next()
	waitUntilNumRenewAttemptsReaches(t, w, 2)

	for i := 0; i < 100; i++ {
		w.Next()
	}
	if atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 2 {
		t.Fatal(`atomic.LoadInt64(&w.Stats.NumRenewAttempts) != 2`)
	}
	if atomic.LoadInt64(&w.Stats.NumRenewed) != 0 {
		t.Fatal(`atomic.LoadInt64(&w.Stats.NumRenewed) != 0`)
	}

	var num int
	w.Scavenger().Filter(func(level, msg string) bool {
		if level == slog.LevelWarn && strings.Contains(msg, "renew failed") && strings.Contains(msg, "foo") {
			num++
		}
		return true
	})
	if num != 2 {
		t.Fatal(`num != 2`)
	}
}

func TestWUID_Step(t *testing.T) {
	const step = 16
	w := NewWUID("alpha", slog.NewScavenger(), WithStep(step, 0))
	w.Reset(17 << 36)

	w.Renew = func() error {
		w.Reset(((atomic.LoadInt64(&w.N) >> 36) + 1) << 36)
		return nil
	}

	for i := int64(1); i < 100; i++ {
		if w.Next()&L36Mask != step*i {
			t.Fatal("w.Next()&L36Mask != step*i")
		}
	}

	n1 := w.Next()
	w.Reset(((n1 >> 36 << 36) | Bye) & ^(step - 1))
	w.Next()
	waitUntilNumRenewedReaches(t, w, 1)
	n2 := w.Next()

	w.Reset(((n2 >> 36 << 36) | Bye) & ^(step - 1))
	w.Next()
	waitUntilNumRenewedReaches(t, w, 2)
	n3 := w.Next()

	if n2>>36-n1>>36 != 1 || n3>>36-n2>>36 != 1 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}

	var num int
	w.Scavenger().Filter(func(level, msg string) bool {
		if level == slog.LevelInfo && strings.Contains(msg, "renew succeeded") {
			num++
		}
		return true
	})
	if num != 2 {
		t.Fatal(`num != 2`)
	}

	func() {
		defer func() {
			_ = recover()
		}()
		NewWUID("alpha", nil, WithStep(5, 0))
		t.Fatal("WithStep should have panicked")
	}()
}

func TestWUID_Floor(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	allSteps := []int64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for loop := 0; loop < 10000; loop++ {
		step := allSteps[r.Intn(len(allSteps))]
		var floor = r.Int63n(step)
		w := NewWUID("alpha", slog.NewScavenger(), WithStep(step, floor))
		if floor < 2 {
			if w.Flags != 0 {
				t.Fatal(`w.Flags != 0`)
			}
		} else {
			if w.Flags != 2 {
				t.Fatal(`w.Flags != 2`)
			}
		}

		w.Reset(r.Int63n(100) << 36)
		baseValue := atomic.LoadInt64(&w.N)

		for i := int64(1); i < 100; i++ {
			x := w.Next()
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

	func() {
		defer func() {
			_ = recover()
		}()
		NewWUID("alpha", nil, WithStep(1024, 2000))
		t.Fatal("WithStep should have panicked")
	}()

	func() {
		defer func() {
			_ = recover()
		}()
		NewWUID("alpha", nil, WithStep(1024, 0), WithStep(128, 0))
		t.Fatal("WithStep should have panicked")
	}()
}

func TestWUID_VerifyH28(t *testing.T) {
	w1 := NewWUID("alpha", nil)
	w1.Reset(H28Mask)
	if err := w1.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 100, error: %s", err)
	}
	if err := w1.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0")
	}
	if err := w1.VerifyH28(0x08000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0x08000000")
	}
	if err := w1.VerifyH28(0x07FFFFFF); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. n: 0x07FFFFFF")
	}

	w2 := NewWUID("alpha", nil, WithSection(1))
	w2.Reset(H28Mask)
	if err := w2.VerifyH28(100); err != nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 100, error: %s", err)
	}
	if err := w2.VerifyH28(0); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0")
	}
	if err := w2.VerifyH28(0x01000000); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0x01000000")
	}
	if err := w2.VerifyH28(0x00FFFFFF); err == nil {
		t.Fatalf("VerifyH28 does not work as expected. section: 1, n: 0x00FFFFFF")
	}
}

func TestWithSection_Panic(t *testing.T) {
	for i := -100; i <= 100; i++ {
		func(j int8) {
			defer func() {
				_ = recover()
			}()
			WithSection(j)
			if j >= 8 {
				t.Fatalf("WithSection should only accept the values in [0, 7]. j: %d", j)
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
				w := NewWUID("alpha", nil, WithSection(j))
				w.Reset(n)
				v := atomic.LoadInt64(&w.N)
				if v>>60 != int64(j) {
					t.Fatalf("w.Section does not work as expected. w.N: %x, n: %x, i: %d, j: %d", v, n, i, j)
				}
			}
		}()
	}

	func() {
		defer func() {
			_ = recover()
		}()
		w := NewWUID("alpha", nil)
		w.Reset((1 << 36) | PanicValue)
		t.Fatal("Reset should have panicked")
	}()
}

func TestWithH28Verifier(t *testing.T) {
	w := NewWUID("alpha", nil, WithH28Verifier(func(h28 int64) error {
		if h28 >= 20 {
			return errors.New("bomb")
		}
		return nil
	}))
	if err := w.VerifyH28(10); err != nil {
		t.Fatal("the H28Verifier should not return error")
	}
	if err := w.VerifyH28(20); err == nil || err.Error() != "bomb" {
		t.Fatal("the H28Verifier was not called")
	}
}

//gocyclo:ignore
func TestWithObfuscation(t *testing.T) {
	w1 := NewWUID("alpha", nil, WithObfuscation(1))
	if w1.Flags != 1 {
		t.Fatal(`w1.Flags != 1`)
	}
	if w1.ObfuscationMask == 0 {
		t.Fatal(`w1.ObfuscationMask == 0`)
	}

	w1.Reset(1 << 36)
	for i := 1; i < 100; i++ {
		v := w1.Next()
		if v&H28Mask != 1<<36 {
			t.Fatal(`v&H28Mask != 1<<36`)
		}
		tmp := v ^ w1.ObfuscationMask
		if tmp&L36Mask != int64(i) {
			t.Fatal(`tmp&L36Mask != int64(i)`)
		}
	}

	w2 := NewWUID("alpha", nil, WithObfuscation(1), WithStep(128, 100))
	if w2.Flags != 3 {
		t.Fatal(`w2.Flags != 3`)
	}
	if w2.ObfuscationMask == 0 {
		t.Fatal(`w2.ObfuscationMask == 0`)
	}

	w2.Reset(1 << 36)
	for i := 1; i < 100; i++ {
		v := w2.Next()
		if v%w2.Floor != 0 {
			t.Fatal(`v%w2.Floor != 0`)
		}
		if v&H28Mask != 1<<36 {
			t.Fatal(`v&H28Mask != 1<<36`)
		}
		tmp := v ^ w2.ObfuscationMask
		if tmp&L36Mask&^(w2.Step-1) != w2.Step*int64(i) {
			t.Fatal(`tmp&L36Mask&^(w2.Step-1) != w2.Step*int64(i)`)
		}
	}

	w3 := NewWUID("alpha", nil, WithObfuscation(1), WithStep(1024, 659))
	if w3.Flags != 3 {
		t.Fatal(`w3.Flags != 3`)
	}
	if w3.ObfuscationMask == 0 {
		t.Fatal(`w3.ObfuscationMask == 0`)
	}

	w3.Reset(1<<36 + 1)
	for i := 1; i < 100; i++ {
		v := w3.Next()
		if v%w3.Floor != 0 {
			t.Fatal(`v%w3.Floor != 0`)
		}
		if v&H28Mask != 1<<36 {
			t.Fatal(`v&H28Mask != 1<<36`)
		}
		tmp := v ^ w3.ObfuscationMask
		if tmp&L36Mask&^(w3.Step-1) != w3.Step*int64(i+1) {
			t.Fatal(`tmp&L36Mask&^(w3.Step-1) != w3.Step*int64(i+1)`)
		}
	}

	func() {
		defer func() {
			_ = recover()
		}()
		NewWUID("alpha", nil, WithObfuscation(0))
		t.Fatal("WithObfuscation should have panicked")
	}()
}
