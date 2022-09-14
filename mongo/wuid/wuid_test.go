package wuid

import (
	"context"
	"errors"
	"fmt"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	dumb = slog.NewDumbLogger()
)

var (
	cfg struct {
		addr   string
		dbName string
		coll   string
		docID  string
	}
)

func init() {
	cfg.addr = "127.0.0.1:27017"
	cfg.dbName = "test"
	cfg.coll = "wuid"
	cfg.docID = "default"
}

func connectMongodb() (*mongo.Client, error) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel1()
	uri := fmt.Sprintf("mongodb://%s", cfg.addr)
	return mongo.Connect(ctx1, options.Client().ApplyURI(uri))
}

func TestWUID_LoadH28FromMongo(t *testing.T) {
	newClient := func() (*mongo.Client, bool, error) {
		client, err := connectMongodb()
		return client, true, err
	}

	w := NewWUID(cfg.docID, dumb)
	err := w.LoadH28FromMongo(newClient, cfg.dbName, cfg.coll, cfg.docID)
	if err != nil {
		t.Fatal(err)
	}

	initial := atomic.LoadInt64(&w.w.N)
	for i := 1; i < 100; i++ {
		if err := w.RenewNow(); err != nil {
			t.Fatal(err)
		}
		expected := ((initial >> 36) + int64(i)) << 36
		if atomic.LoadInt64(&w.w.N) != expected {
			t.Fatalf("w.w.N is %d, while it should be %d. i: %d", atomic.LoadInt64(&w.w.N), expected, i)
		}
		n := rand.Intn(10)
		for j := 0; j < n; j++ {
			w.Next()
		}
	}
}

func TestWUID_LoadH28FromMongo_Error(t *testing.T) {
	w := NewWUID(cfg.docID, dumb)
	if w.LoadH28FromMongo(nil, "", cfg.coll, cfg.docID) == nil {
		t.Fatal("dbName is not properly checked")
	}
	if w.LoadH28FromMongo(nil, cfg.dbName, "", cfg.docID) == nil {
		t.Fatal("coll is not properly checked")
	}
	if w.LoadH28FromMongo(nil, cfg.dbName, cfg.coll, "") == nil {
		t.Fatal("docID is not properly checked")
	}

	newErrorClient := func() (*mongo.Client, bool, error) {
		return nil, true, errors.New("beta")
	}
	if w.LoadH28FromMongo(newErrorClient, cfg.dbName, cfg.coll, cfg.docID) == nil {
		t.Fatal(`w.LoadH28FromMongo(newErrorClient, cfg.dbName, cfg.coll, cfg.docID) == nil`)
	}
}

func waitUntilNumRenewedReaches(t *testing.T, w *WUID, expected int64) {
	t.Helper()
	startTime := time.Now()
	for time.Since(startTime) < time.Second*3 {
		if atomic.LoadInt64(&w.w.Stats.NumRenewed) == expected {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatal("timeout")
}

func TestWUID_Renew(t *testing.T) {
	client, err := connectMongodb()
	if err != nil {
		t.Fatal(err)
	}
	newClient := func() (*mongo.Client, bool, error) {
		return client, false, err
	}

	w := NewWUID(cfg.docID, slog.NewScavenger())
	err = w.LoadH28FromMongo(newClient, cfg.dbName, cfg.coll, cfg.docID)
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
	newClient := func() (*mongo.Client, bool, error) {
		var client *mongo.Client
		// ...
		return client, true, nil
	}

	// Setup
	w := NewWUID("alpha", nil)
	err := w.LoadH28FromMongo(newClient, "test", "wuid", "default")
	if err != nil {
		panic(err)
	}

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", w.Next())
	}
}
