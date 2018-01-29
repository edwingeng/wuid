package wuid

import (
	"math/rand"
	"sync/atomic"
	"testing"
)

func getMysqlConfig() (string, string, string, string, string) {
	return "127.0.0.1:3306", "root", "", "test", "wuid"
}

func TestWUID_LoadH24FromMysql(t *testing.T) {
	var nextValue uint64
	g := NewWUID("default", nil)
	for i := 0; i < 1000; i++ {
		err := g.LoadH24FromMysql(getMysqlConfig())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextValue = atomic.LoadUint64(&g.w.N)
		} else {
			nextValue = ((nextValue >> 40) + 1) << 40
		}
		if atomic.LoadUint64(&g.w.N) != nextValue {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), nextValue, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}
