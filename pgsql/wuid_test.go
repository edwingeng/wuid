package wuid

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/wuid/internal" // use of internal package discouraged
	_ "github.com/lib/pq"                // postgres driver
)

type simpleLogger struct{}
type config struct {
	host  string
	port  int
	user  string
	pass  string
	db    string
	table string
}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

// Test database config
var pgc = &config{
	host:  "localhost",
	port:  5432,
	user:  "postgres",
	pass:  "mysecretpassword",
	db:    "postgres",
	table: "wuid",
}

// Create table in test database
func init() {
	// create db table for testing
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", pgc.host, pgc.user, pgc.pass, pgc.db)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("pgsql init failed error: ", err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE wuid
		(
			h serial NOT NULL UNIQUE,
			x int NOT NULL PRIMARY KEY DEFAULT '0'
		)`)
	if err != nil {
		fmt.Println("pgsql table creation error, this is expected if table already created. error: ", err)
	}
}
func TestLoadH24FromPg(t *testing.T) {
	var nextVal uint64
	g := NewWUID("default", sl)
	for i := 0; i < 1000; i++ {
		err := g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextVal = atomic.LoadUint64(&g.w.N)
		} else {
			nextVal = ((nextVal >> 40) + 1) << 40
		}
		if atomic.LoadUint64(&g.w.N) != nextVal {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), nextVal, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}

	// Check proper connection fail response
	err := g.LoadH24FromPg(pgc.host, "badusername", pgc.pass, pgc.db, pgc.table)
	if err == nil {
		t.Fatal("Connection should fail and return error")
	}

	// Check connection parameter validation

	if g.LoadH24FromPg("", pgc.user, pgc.pass, pgc.db, pgc.table) == nil {
		t.Fatal("host is not properly checked")
	}

	if g.LoadH24FromPg(pgc.host, "", pgc.pass, pgc.db, pgc.table) == nil {
		t.Fatal("user is not properly checked")
	}

	if g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, "", pgc.table) == nil {
		t.Fatal("db name is not properly checked")
	}

	if g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, "") == nil {
		t.Fatal("table is not properly checked")
	}

	if g.LoadH24FromPg("127.0.0.1:30000", pgc.user, pgc.pass, pgc.db, "") == nil {
		t.Fatal("LoadH24FromPg should fail when host is invalid")
	}

	fmt.Println(" - " + t.Name() + " complete - ")
}

func TestLoadH24FromPgWithOpts(t *testing.T) {
	// Setup
	var nextVal uint64
	g := NewWUID("default", sl)

	// Test expected successful connection, no SSL/TLS
	for i := 0; i < 500; i++ {
		err := g.LoadH24FromPgWithOpts(pgc.host, pgc.port, pgc.user, pgc.pass, pgc.db, pgc.table, "disable", 5, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			nextVal = atomic.LoadUint64(&g.w.N)
		} else {
			nextVal = ((nextVal >> 40) + 1) << 40
		}
		if atomic.LoadUint64(&g.w.N) != nextVal {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), nextVal, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}

	// Check connection parameter validation

	if g.LoadH24FromPgWithOpts("", pgc.port, pgc.user, pgc.pass, pgc.db, pgc.table, "disable", 5, "", "", "") == nil {
		t.Fatal("host is not properly checked")
	}

	if g.LoadH24FromPgWithOpts(pgc.host, pgc.port, "", pgc.pass, pgc.db, pgc.table, "disable", 5, "", "", "") == nil {
		t.Fatal("user is not properly checked")
	}

	if g.LoadH24FromPgWithOpts(pgc.host, pgc.port, pgc.user, pgc.pass, "", pgc.table, "disable", 5, "", "", "") == nil {
		t.Fatal("db name is not properly checked")
	}

	if g.LoadH24FromPgWithOpts(pgc.host, pgc.port, pgc.user, pgc.pass, pgc.db, "", "disable", 5, "", "", "") == nil {
		t.Fatal("table is not properly checked")
	}

	if g.LoadH24FromPgWithOpts("127.0.0.1:300000", pgc.port, pgc.user, pgc.pass, pgc.db, pgc.table, "disable", 5, "", "", "") == nil {
		t.Fatal("LoadH24FromPg should fail when host is invalid")
	}

	fmt.Println(" - " + t.Name() + " complete - ")
}

func TestWUID_LoadH24FromPg_UserPass(t *testing.T) {
	var err error
	g := NewWUID("default", sl)
	err = g.LoadH24FromPg(pgc.host, "wuid", "abc123", pgc.db, pgc.table)
	if err != nil {
		if strings.Contains(err.Error(), "authentication failed for user") {
			t.Log("you need to create a user in your Postgres. username: wuid, password: abc123")
		} else {
			t.Fatal(err)
		}
	}
	err = g.LoadH24FromPg(pgc.host, "wuid", "nopass", pgc.db, pgc.table)
	if err == nil {
		t.Fatal("LoadH24FromPg should fail when the password is incorrect")
	}

	fmt.Println(" - " + t.Name() + " complete - ")
}

func TestWUID_Next_Renew(t *testing.T) {
	g := NewWUID("default", sl)
	err := g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)
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

	fmt.Println(" - " + t.Name() + " complete - ")
}

func TestWithSection(t *testing.T) {
	g := NewWUID("default", sl, WithSection(15))
	err := g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 15 {
		t.Fatal("WithSection does not work as expected")
	}

	fmt.Println(" - " + t.Name() + " complete - ")
}

func BenchmarkLoadH24FromPg(b *testing.B) {
	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)

	//Generate
	for n := 0; n < b.N; n++ {
		g.Next()
	}

	fmt.Println(" - " + b.Name() + " complete - ")
}
