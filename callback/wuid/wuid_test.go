package wuid

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/edwingeng/slog"
)

func TestWUID_LoadH28WithCallback_Error(t *testing.T) {
	var err error
	g := NewWUID("default", slog.NewDumbLogger())
	err = g.LoadH28WithCallback(nil)
	if err == nil {
		t.Fatal("LoadH28WithCallback should fail when cb is nil")
	}

	err = g.LoadH28WithCallback(func() (uint64, func(), error) {
		return 0, nil, errors.New("foo")
	})
	if err == nil {
		t.Fatal("LoadH28WithCallback should fail when cb returns an error")
	}

	err = g.LoadH28WithCallback(func() (uint64, func(), error) {
		return 0, nil, nil
	})
	if err == nil {
		t.Fatal("LoadH28WithCallback should fail when cb returns an invalid h28")
	}
}

func TestWUID_LoadH28WithCallback(t *testing.T) {
	var h28, counter uint64
	done := func() {
		counter++
	}
	cb := func() (uint64, func(), error) {
		return atomic.AddUint64(&h28, 1), done, nil
	}

	g := NewWUID("default", slog.NewDumbLogger())
	for i := 0; i < 1000; i++ {
		err := g.LoadH28WithCallback(cb)
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 36
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}

	if counter != 1000 {
		t.Fatalf("the callback done do not work as expected. counter: %d", counter)
	}
}

func TestWUID_LoadH28WithCallback_Section(t *testing.T) {
	var h28 uint64
	cb := func() (uint64, func(), error) {
		return atomic.AddUint64(&h28, 1), nil, nil
	}

	g := NewWUID("default", slog.NewDumbLogger(), WithSection(1))
	for i := 0; i < 1000; i++ {
		err := g.LoadH28WithCallback(cb)
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1 + 0x1000000) << 36
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH28WithCallback_Same(t *testing.T) {
	cb := func() (uint64, func(), error) {
		return 100, nil, nil
	}

	g1 := NewWUID("default", slog.NewDumbLogger())
	_ = g1.LoadH28WithCallback(cb)
	if err := g1.LoadH28WithCallback(cb); err == nil {
		t.Fatal("LoadH28WithCallback should return an error")
	}

	g2 := NewWUID("default", slog.NewDumbLogger(), WithSection(1))
	_ = g2.LoadH28WithCallback(cb)
	if err := g2.LoadH28WithCallback(cb); err == nil {
		t.Fatal("LoadH28WithCallback should return an error")
	}
}

func Example() {
	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH28WithCallback(func() (uint64, func(), error) {
		resp, err := http.Get("https://stackoverflow.com/")
		if resp != nil {
			defer func() {
				_ = resp.Body.Close()
			}()
		}
		if err != nil {
			return 0, nil, err
		}

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, nil, err
		}

		fmt.Printf("Page size: %d (%#06x)\n\n", len(bytes), len(bytes))
		return uint64(len(bytes)), nil, nil
	})

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
