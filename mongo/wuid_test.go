package wuid

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/wuid/internal"
)

func getMongoConfig() (string, string, string, string, string, string) {
	return "127.0.0.1:27017", "", "", "test", "foo", "wuid"
}

func TestWUID_LoadH24FromMongo(t *testing.T) {
	var nextValue uint64
	g := NewWUID("default", nil)
	for i := 0; i < 1000; i++ {
		err := g.LoadH24FromMongo(getMongoConfig())
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

func TestWUID_LoadH24FromMongo_Error(t *testing.T) {
	g := NewWUID("default", nil)
	addr, user, pass, dbName, coll, docID := getMongoConfig()

	if g.LoadH24FromMongo("", user, pass, dbName, coll, docID) == nil {
		t.Fatal("addr is not properly checked")
	}
	if g.LoadH24FromMongo(addr, user, pass, "", coll, docID) == nil {
		t.Fatal("dbName is not properly checked")
	}
	if g.LoadH24FromMongo(addr, user, pass, dbName, "", docID) == nil {
		t.Fatal("coll is not properly checked")
	}
	if g.LoadH24FromMongo(addr, user, pass, dbName, coll, "") == nil {
		t.Fatal("docID is not properly checked")
	}

	if g.LoadH24FromMongoWithTimeout("127.0.0.1:30000", user, pass, dbName, coll, docID, time.Second) == nil {
		t.Fatal("LoadH24FromMongoWithTimeout should fail when is address is invalid")
	}
}

func TestWUID_LoadH24FromMongo_UserPass(t *testing.T) {
	var err error
	g := NewWUID("default", nil)
	addr, _, _, dbName, coll, docID := getMongoConfig()
	err = g.LoadH24FromMongo(addr, "wuid", "abc123", dbName, coll, docID)
	if err != nil {
		if strings.Contains(err.Error(), "Authentication failed") {
			t.Log("you need to create a user in your Mongo. username: wuid, password: abc123")
		} else {
			t.Fatal(err)
		}
	}
	err = g.LoadH24FromMongo(addr, "wuid", "nopass", dbName, coll, docID)
	if err == nil {
		t.Fatal("LoadH24FromMongo should fail when the password is incorrect")
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	g := NewWUID("default", nil)
	err := g.LoadH24FromMongo(getMongoConfig())
	if err != nil {
		t.Fatal(err)
	}

	n1 := g.Next()
	kk := ((internal.CriticalValue + internal.RenewInterval) & ^internal.RenewInterval) - 1

	g.w.Reset((n1 >> 40 << 40) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.w.Reset((n2 >> 40 << 40) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>40 == n1>>40 || n3>>40 == n2>>40 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>40, n2>>40, n3>>40)
	}
}

func TestWithSection(t *testing.T) {
	g := NewWUID("default", nil, WithSection(15))
	err := g.LoadH24FromMongo(getMongoConfig())
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 15 {
		t.Fatal("WithSection does not work as expected")
	}
}

func Example() {
	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH24FromMongo("127.0.0.1:27017", "", "", "test", "foo", "wuid")

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
