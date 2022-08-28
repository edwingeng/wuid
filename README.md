# Overview
- WUID is a globally unique number generator.
- WUID is **10-135** times faster than UUID and thousands of times faster than generating unique numbers with Redis.
- In the nutshell, WUID generates unique 64-bit integers in sequence. The high 28 bits are loaded from a data store. By now, Redis, MySQL, MongoDB and Callback are supported.

# Benchmarks
```
BenchmarkWUID           159393580          7.661 ns/op        0 B/op       0 allocs/op
BenchmarkRand           100000000         14.95 ns/op         0 B/op       0 allocs/op
BenchmarkTimestamp      164224915          7.359 ns/op        0 B/op       0 allocs/op
BenchmarkUUID_V1        23629536          43.42 ns/op         0 B/op       0 allocs/op
BenchmarkUUID_V2        29351550          43.96 ns/op         0 B/op       0 allocs/op
BenchmarkUUID_V3         4703044         254.2 ns/op        144 B/op       4 allocs/op
BenchmarkUUID_V4         5796310         210.0 ns/op         16 B/op       1 allocs/op
BenchmarkUUID_V5         4051291         310.7 ns/op        168 B/op       4 allocs/op
BenchmarkRedis              2996       38725 ns/op          160 B/op       5 allocs/op
BenchmarkSnowflake       1000000        2092 ns/op            0 B/op       0 allocs/op
BenchmarkULID            5660170         207.7 ns/op         16 B/op       1 allocs/op
BenchmarkXID            49639082          26.21 ns/op         0 B/op       0 allocs/op
BenchmarkShortID         1312386         922.2 ns/op        320 B/op      11 allocs/op
BenchmarkKsuid          19717675          59.79 ns/op         0 B/op       0 allocs/op
```

# Features
- Extremely fast
- Lock free
- Being unique across time
- Being unique within a data center
- Being unique globally if all data centers share a same data store, or they use different section IDs
- Being capable of generating 100M unique numbers in a single second with each WUID instance
- Auto-renew when the low 36 bits are about to run out

# Install
``` bash
go get -u github.com/edwingeng/wuid
```

# Usage examples
### Redis
``` go
import "github.com/edwingeng/wuid/redis/wuid"

newClient := func() (redis.UniversalClient, bool, error) {
    var client redis.UniversalClient
    // ...
    return client, true, nil
}

// Setup
g := NewWUID("default", nil)
_ = g.LoadH28FromRedis(newClient, "wuid")

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", g.Next())
}
```

### MySQL
``` go
import "github.com/edwingeng/wuid/mysql/wuid"

newDB := func() (*sql.DB, bool, error) {
    var db *sql.DB
    // ...
    return db, true, nil
}

// Setup
g := NewWUID("default", nil)
_ = g.LoadH28FromMysql(newDB, "wuid")

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", g.Next())
}
```

### MongoDB
``` go
import "github.com/edwingeng/wuid/mongo/wuid"

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
```

### Callback
``` go
import "github.com/edwingeng/wuid/callback/wuid"

// Setup
g := NewWUID("default", nil)
_ = g.LoadH28WithCallback(func() (int64, func(), error) {
    resp, err := http.Get("https://stackoverflow.com/")
    if resp != nil {
        defer func() {
            _ = resp.Body.Close()
        }()
    }
    if err != nil {
        return 0, nil, err
    }

    bytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return 0, nil, err
    }

    fmt.Printf("Page size: %d (%#06x)\n\n", len(bytes), len(bytes))
    return int64(len(bytes)), nil, nil
})

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", g.Next())
}
```

# Mysql table creation
``` sql
CREATE TABLE IF NOT EXISTS `wuid` (
    `h` int(10) NOT NULL AUTO_INCREMENT,
    `x` tinyint(4) NOT NULL DEFAULT '0',
    PRIMARY KEY (`x`),
    UNIQUE KEY `h` (`h`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
```

# Section ID
You can specify a custom section ID for the generated numbers with `wuid.WithSection` when you call `wuid.NewWUID`. The section ID must be in between `[0, 7]`.

# Step
You can customize the step value of `Next()` with `wuid.WithStep`.

# Best practices
- Pass a logger to `wuid.NewWUID` and keep an eye on the warnings that include "renew failed", which means that the low 36 bits are about to run out in hours to hundreds of hours, and WUID fails to get a new number from your data store.

# Special thanks
- [dustinfog](https://github.com/dustinfog)
