# Overview
- WUID is a globally unique number generator. It is tens of times faster than UUID. Each instance can even generate 100M unique numbers in a single second.
- In the nutshell, WUID generates unique 64-bit integers in sequence. The high 28 bits are loaded from a data store. By now, Redis, MySQL, MongoDB and Callback are supported.
- WUID renews the high 28 bits automatically when the low 36 bits are about to run out.
- WUID guarantees the uniqueness as long as all its instances share a same data store.
- Number obfuscation is supported.
- WUID is lock free.

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

# Getting Started
``` bash
go get -u github.com/edwingeng/wuid
```

# Usages
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

# Mysql Table Creation
``` sql
CREATE TABLE IF NOT EXISTS `wuid` (
    `h` int(10) NOT NULL AUTO_INCREMENT,
    `x` tinyint(4) NOT NULL DEFAULT '0',
    PRIMARY KEY (`x`),
    UNIQUE KEY `h` (`h`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
```

# Options

- `WithObfuscation` enables number obfuscation.
- `WithStep` sets the step and the floor for each number.
- `WithSection` brands a section ID on each number. A section ID must be in between [0, 7].

# Attentions
It is highly recommended to pass a logger to `wuid.NewWUID` and keep an eye on the warnings that include "renew failed", which indicates that the low 36 bits are about to run out in hours to hundreds of hours, and WUID failed to get a new number from your data store. But don't worry too much, WUID will make an attempt once in a while. 

# Special thanks
- [dustinfog](https://github.com/dustinfog)
