package wuid

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync/atomic"
	"testing"
)

type simpleLogger struct{}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

func TestWUID_LoadH24WithCallback_Error(t *testing.T) {
	var err error
	g := NewWUID("default", sl)
	err = g.LoadH24WithCallback(nil)
	if err == nil {
		t.Fatal("LoadH24WithCallback should fail when cb is nil")
	}

	err = g.LoadH24WithCallback(func() (uint64, error) {
		return 0, errors.New("foo")
	})
	if err == nil {
		t.Fatal("LoadH24WithCallback should fail when cb returns an error")
	}

	err = g.LoadH24WithCallback(func() (uint64, error) {
		return 0, nil
	})
	if err == nil {
		t.Fatal("LoadH24WithCallback should fail when cb returns an invalid h24")
	}
}

func TestWUID_LoadH24WithCallback(t *testing.T) {
	var h24 uint64
	cb := func() (uint64, error) {
		return atomic.AddUint64(&h24, 1), nil
	}

	g := NewWUID("default", sl)
	for i := 0; i < 1000; i++ {
		err := g.LoadH24WithCallback(cb)
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH24WithCallback_Section(t *testing.T) {
	var h24 uint64
	cb := func() (uint64, error) {
		return atomic.AddUint64(&h24, 1), nil
	}

	g := NewWUID("default", sl, WithSection(1))
	for i := 0; i < 1000; i++ {
		err := g.LoadH24WithCallback(cb)
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1 + 0x100000) << 40
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH24WithCallback_Same(t *testing.T) {
	cb := func() (uint64, error) {
		return 100, nil
	}

	g1 := NewWUID("default", sl)
	_ = g1.LoadH24WithCallback(cb)
	if err := g1.LoadH24WithCallback(cb); err == nil {
		t.Fatal("LoadH24WithCallback should return an error")
	}

	g2 := NewWUID("default", sl, WithSection(1))
	_ = g2.LoadH24WithCallback(cb)
	if err := g2.LoadH24WithCallback(cb); err == nil {
		t.Fatal("LoadH24WithCallback should return an error")
	}
}

func Example() {
	g := NewWUID("default", nil)
	_ = g.LoadH24WithCallback(func() (uint64, error) {
		resp, err := http.Get("https://stackoverflow.com/")
		if resp != nil {
			defer func() {
				_ = resp.Body.Close()
			}()
		}
		if err != nil {
			return 0, err
		}

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		fmt.Printf("Page size: %d (%#06x)\n\n", len(bytes), len(bytes))
		return uint64(len(bytes)), nil
	})

	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
