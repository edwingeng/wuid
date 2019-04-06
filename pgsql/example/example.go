package main

import (
	"fmt"

	"github.com/edwingeng/wuid/pgsql"
	_ "github.com/lib/pq" // postgres driver
)

func Example() {
	// Setup
	g := wuid.NewWUID("default", nil)
	_ = g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
