# The postgres version needs to be upgraded according to the new design. Please help if you are familier with postgres.

# Overview

This is a Postgres compatible version of [WUID](https://github.com/edwingeng/wuid).

- WUID is a unique number generator, but it is not a UUID implementation.
- WUID is 10-135 times faster than UUID and 4600 times faster than generating unique numbers with Redis.
- WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store. By now, Redis, MySQL, and MongoDB are supported.

# Install
``` bash
go get -u github.com/edwingeng/wuid/...
```

Or choose one command from the following if you prefer dep:
```bash
dep ensure -add github.com/edwingeng/wuid/pgsql

dep ensure -add github.com/edwingeng/wuid/redis
dep ensure -add github.com/edwingeng/wuid/mysql
dep ensure -add github.com/edwingeng/wuid/mongo
dep ensure -add github.com/edwingeng/wuid/callback
```

**_CockroachDB_**

[CockroachDB](https://www.cockroachlabs.com) uses the same connection driver as PostgreSQL,
but this package has not been tested with CockroachDB yet; however support is planned. I
expected it will not work properly in its current form because CockroachDB ```serial``` data
type is not sequential.

# Usage Examples

### PostgreSQL

> Postgres driver [lib/pq](https://github.com/lib/pq) expects SSL by default. You must use
> LoadH24FromPgWithOpts() if you need to specify connection parameters. This functionality
> was left as a secure default.

```go
    import "github.com/edwingeng/wuid/pgsql"
    
    // Setup
    g := NewWUID("default", nil)
    err := g.LoadH24FromPg(pgc.host, pgc.user, pgc.pass, pgc.db, pgc.table)
    if err != nil {
        t.Fatal(err)
    }

    // Generate
    for i := 0; i < 10; i++ {
        fmt.Printf("%#016x\n", g.Next())
    }
```

### PostgreSQL table creation

```sql
CREATE TABLE wuid
(
    h serial NOT NULL UNIQUE,
    x int NOT NULL PRIMARY KEY DEFAULT '0'
);
```

Test Postgres Docker setup
> This will setup Docker Postgres:Alpine container with TLS, run tests and tear down Docker container.
```bash
# Run from package ../wuid/pgsql directory
## Make executable: 
chmod +x testdb.sh
## Run: 
./testdb.sh
```