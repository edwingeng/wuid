package wuid

import (
	"math/rand"
	"sync/atomic"
	"testing"
)

func getMongoConfig() (string, string, string, string, string, string) {
	return "127.0.0.1:27017", "", "", "test", "foo", "wuid"
}

func TestWUID_LoadH24FromMongo(t *testing.T) {
	var nextValue uint64
	wuid := NewWUID("default", nil)
	for i := 0; i < 100; i++ {
		err := wuid.LoadH24FromMongo(getMongoConfig())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextValue = atomic.LoadUint64(&wuid.w.N)
		} else {
			nextValue = ((nextValue >> 40) + 1) << 40
		}
		if atomic.LoadUint64(&wuid.w.N) != nextValue {
			t.Fatalf("wuid.n is %d, while it should be %d", atomic.LoadUint64(&wuid.w.N), nextValue)
		}
		for j := 0; j < rand.Intn(10); j++ {
			wuid.Next()
		}
	}
}
