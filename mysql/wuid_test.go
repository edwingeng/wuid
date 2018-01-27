package wuid

import (
	"math/rand"
	"testing"
)

func getMysqlConfig() (string, string, string, string, string) {
	return "127.0.0.1:3306", "root", "", "test", "wuid"
}

func TestWUID_LoadH24FromMysql(t *testing.T) {
	var nextValue uint64
	wuid := NewWUID()
	for i := 0; i < 100; i++ {
		err := wuid.LoadH24FromMysql(getMysqlConfig())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextValue = wuid.n
		} else {
			nextValue = ((nextValue >> 40) + 1) << 40
		}
		if wuid.n != nextValue {
			t.Fatalf("wuid.n is %d, while it should be %d", wuid.n, nextValue)
		}
		for j := 0; j < rand.Intn(10); j++ {
			wuid.Next()
		}
	}
}
