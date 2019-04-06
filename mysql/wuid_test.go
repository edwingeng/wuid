package wuid

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/wuid/internal"
	_ "github.com/go-sql-driver/mysql"
)

type simpleLogger struct{}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

func init() {
	addr, user, pass, dbName, table := getMysqlConfig()

	var dsn string
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@tcp(" + addr + ")/" + dbName

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("mysql connection error: ", err)
	}
	defer func() {
		_ = db.Close()
	}()

	_, err = db.Exec(fmt.Sprintf("SELECT 1 FROM %s.%s LIMIT 1", dbName, table))
	if err != nil {
		format := "Table '%s.%s' doesn't exist. You can create it with github.com/edwingeng/wuid/mysql/db.sql"
		panic(fmt.Sprintf(format, dbName, table))
	}
}

func getMysqlConfig() (string, string, string, string, string) {
	return "127.0.0.1:3306", "root", "password", "test", "wuid"
}

func connect(addr, user, pass, dbName string) (*sql.DB, error) {
	var dsn string
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@tcp(" + addr + ")/" + dbName

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestWUID_LoadH24FromMysql(t *testing.T) {
	addr, user, pass, dbName, table := getMysqlConfig()
	db, err := connect(addr, user, pass, dbName)
	if err != nil {
		t.Fatal(err)
	}

	var nextValue uint64
	g := NewWUID("default", sl)
	for i := 0; i < 1000; i++ {
		err := g.LoadH24FromMysql(db, table)
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

func TestWUID_LoadH24FromMysql_Error(t *testing.T) {
	g := NewWUID("default", sl)
	if g.LoadH24FromMysql(nil, "") == nil {
		t.Fatal("table is not properly checked")
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	addr, user, pass, dbName, table := getMysqlConfig()
	db, err := connect(addr, user, pass, dbName)
	if err != nil {
		t.Fatal(err)
	}

	g := NewWUID("default", sl)
	err = g.LoadH24FromMysql(db, table)
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
	addr, user, pass, dbName, table := getMysqlConfig()
	db, err := connect(addr, user, pass, dbName)
	if err != nil {
		t.Fatal(err)
	}

	g := NewWUID("default", sl, WithSection(15))
	err = g.LoadH24FromMysql(db, table)
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 15 {
		t.Fatal("WithSection does not work as expected")
	}
}

func Example() {
	var db *sql.DB

	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH24FromMysql(db, "wuid")

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
