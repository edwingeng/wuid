package wuid

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/wuid/internal"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type simpleLogger struct{}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

func getMongoConfig() (string, string, string, string) {
	return "127.0.0.1:27017", "test", "wuid", "default"
}

func connect(addr string) (*mongo.Client, error) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel1()
	uri := fmt.Sprintf("mongodb://%s", addr)
	return mongo.Connect(ctx1, options.Client().ApplyURI(uri))
}

func TestWUID_LoadH28FromMongo(t *testing.T) {
	addr, dbName, coll, docID := getMongoConfig()
	newClient := func() (*mongo.Client, bool, error) {
		client, err := connect(addr)
		return client, true, err
	}

	var nextValue uint64
	g := NewWUID(docID, sl)
	for i := 0; i < 1000; i++ {
		err := g.LoadH28FromMongo(newClient, dbName, coll, docID)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextValue = atomic.LoadUint64(&g.w.N)
		} else {
			nextValue = ((nextValue >> 36) + 1) << 36
		}
		if atomic.LoadUint64(&g.w.N) != nextValue {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), nextValue, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH28FromMongo_Error(t *testing.T) {
	_, dbName, coll, docID := getMongoConfig()
	g := NewWUID(docID, sl)

	if g.LoadH28FromMongo(nil, "", coll, docID) == nil {
		t.Fatal("dbName is not properly checked")
	}
	if g.LoadH28FromMongo(nil, dbName, "", docID) == nil {
		t.Fatal("coll is not properly checked")
	}
	if g.LoadH28FromMongo(nil, dbName, coll, "") == nil {
		t.Fatal("docID is not properly checked")
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	addr, dbName, coll, docID := getMongoConfig()
	client, err := connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	newClient := func() (*mongo.Client, bool, error) {
		return client, false, nil
	}

	g := NewWUID(docID, sl)
	err = g.LoadH28FromMongo(newClient, dbName, coll, docID)
	if err != nil {
		t.Fatal(err)
	}

	n1 := g.Next()
	kk := ((internal.CriticalValue + internal.RenewInterval) & ^internal.RenewInterval) - 1

	g.w.Reset((n1 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.w.Reset((n2 >> 36 << 36) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>36 == n1>>36 || n3>>36 == n2>>36 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>36, n2>>36, n3>>36)
	}
}

func TestWithSection(t *testing.T) {
	addr, dbName, coll, docID := getMongoConfig()
	client, err := connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	newClient := func() (*mongo.Client, bool, error) {
		return client, false, nil
	}

	g := NewWUID(docID, sl, WithSection(15))
	err = g.LoadH28FromMongo(newClient, dbName, coll, docID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 15 {
		t.Fatal("WithSection does not work as expected")
	}
}

func Example() {
	newClient := func() (*mongo.Client, bool, error) {
		var client *mongo.Client
		// ...
		return client, true, nil
	}

	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH28FromMongo(newClient, "test", "wuid", "default")

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
