package wuid

import (
	"errors"
	"fmt"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	dumb = slog.NewDumbLogger()
)

func TestWUID_LoadH28WithCallback_Error(t *testing.T) {
	w := NewWUID("alpha", dumb)
	err1 := w.LoadH28WithCallback(nil)
	if err1 == nil {
		t.Fatal("LoadH28WithCallback should fail when cb is nil")
	}

	err2 := w.LoadH28WithCallback(func() (int64, func(), error) {
		return 0, nil, errors.New("foo")
	})
	if err2 == nil {
		t.Fatal("LoadH28WithCallback should fail when cb returns an error")
	}

	err3 := w.LoadH28WithCallback(func() (int64, func(), error) {
		return 0, nil, nil
	})
	if err3 == nil {
		t.Fatal("LoadH28WithCallback should fail when cb returns an invalid h28")
	}
}

func TestWUID_LoadH28WithCallback(t *testing.T) {
	var h28, counter int64
	done := func() {
		counter++
	}
	cb := func() (int64, func(), error) {
		return atomic.AddInt64(&h28, 1), done, nil
	}

	w := NewWUID("alpha", dumb)
	err := w.LoadH28WithCallback(cb)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 1000; i++ {
		if err := w.RenewNow(); err != nil {
			t.Fatal(err)
		}
		v := (int64(i) + 1) << 36
		if atomic.LoadInt64(&w.w.N) != v {
			t.Fatalf("w.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&w.w.N), v, i)
		}
		n := rand.Intn(10)
		for j := 0; j < n; j++ {
			w.Next()
		}
	}

	if counter != 1000 {
		t.Fatalf("the callback done do not work as expected. counter: %d", counter)
	}
}

func TestWUID_LoadH28WithCallback_Section(t *testing.T) {
	var h28 int64
	cb := func() (int64, func(), error) {
		return atomic.AddInt64(&h28, 1), nil, nil
	}

	w := NewWUID("alpha", dumb, WithSection(1))
	for i := 0; i < 1000; i++ {
		err := w.LoadH28WithCallback(cb)
		if err != nil {
			t.Fatal(err)
		}
		v := (int64(i) + 1 + 0x1000000) << 36
		if atomic.LoadInt64(&w.w.N) != v {
			t.Fatalf("w.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&w.w.N), v, i)
		}
		n := rand.Intn(10)
		for j := 0; j < n; j++ {
			w.Next()
		}
	}
}

func TestWUID_LoadH28WithCallback_Same(t *testing.T) {
	cb := func() (int64, func(), error) {
		return 100, nil, nil
	}

	w1 := NewWUID("alpha", dumb)
	_ = w1.LoadH28WithCallback(cb)
	if err := w1.LoadH28WithCallback(cb); err == nil {
		t.Fatal("LoadH28WithCallback should return an error")
	}

	w2 := NewWUID("alpha", dumb, WithSection(1))
	_ = w2.LoadH28WithCallback(cb)
	if err := w2.LoadH28WithCallback(cb); err == nil {
		t.Fatal("LoadH28WithCallback should return an error")
	}
}

func waitUntilNumRenewedReaches(t *testing.T, w *WUID, expected int64) {
	t.Helper()
	startTime := time.Now()
	for time.Since(startTime) < time.Second {
		if atomic.LoadInt64(&w.w.Stats.NumRenewed) == expected {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("timeout")
}

func TestWUID_Renew(t *testing.T) {
	w := NewWUID("alpha", slog.NewScavenger())
	err := w.LoadH28WithCallback(func() (h28 int64, clean func(), err error) {
		return (atomic.LoadInt64(&w.w.N) >> 36) + 1, nil, nil
	})
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
	callback := func() (int64, func(), error) {
		var h28 int64
		// ...
		return h28, nil, nil
	}

	// Setup
	w := NewWUID("alpha", nil)
	err := w.LoadH28WithCallback(callback)
	if err != nil {
		panic(err)
	}

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", w.Next())
	}
}
