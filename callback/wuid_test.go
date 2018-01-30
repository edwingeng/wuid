package wuid

import (
	"errors"
	"math/rand"
	"sync/atomic"
	"testing"
)

func TestWUID_LoadH24WithCallback_Error(t *testing.T) {
	g := NewWUID("default", nil)
	err := g.LoadH24WithCallback(func() (uint64, error) {
		return 0, errors.New("foo")
	})
	if err == nil {
		t.Fatal("LoadH24WithCallback should fail when cb returns an error")
	}
}

func TestWUID_LoadH24WithCallback(t *testing.T) {
	var h24 uint64
	cb := func() (uint64, error) {
		return atomic.AddUint64(&h24, 1), nil
	}

	g := NewWUID("default", nil)
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

	g := NewWUID("default", nil, WithSection(1))
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

	g1 := NewWUID("default", nil)
	_ = g1.LoadH24WithCallback(cb)
	if err := g1.LoadH24WithCallback(cb); err == nil {
		t.Fatal("LoadH24WithCallback should return an error")
	}

	g2 := NewWUID("default", nil, WithSection(1))
	_ = g2.LoadH24WithCallback(cb)
	if err := g2.LoadH24WithCallback(cb); err == nil {
		t.Fatal("LoadH24WithCallback should return an error")
	}
}
