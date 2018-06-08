# Overview

# Install
``` bash
go get -u github.com/edwingeng/wuid/...
```

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