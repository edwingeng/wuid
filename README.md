# Overview
- `WUID` is a universal unique identifier generator.
- `WUID` is much faster than traditional UUID. Each `WUID` instance can even generate 100M unique identifiers in a single second.
- In the nutshell, `WUID` generates 64-bit integers in sequence. The high 28 bits are loaded from a data source. By now, Redis, MySQL, MongoDB and Callback are supported.
- The uniqueness is guaranteed as long as all `WUID` instances share a same data source or each group of them has a different section ID.
- `WUID` automatically renews the high 28 bits when the low 36 bits are about to run out.
- `WUID` is thread-safe, and lock free.
- Obfuscation is supported.

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
import "github.com/edwingeng/wuid/redis/v8/wuid"

newClient := func() (redis.UniversalClient, bool, error) {
    var client redis.UniversalClient
    // ...
    return client, true, nil
}

// Setup
w := NewWUID("alpha", nil)
err := w.LoadH28FromRedis(newClient, "wuid")
if err != nil {
    panic(err)
}

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", w.Next())
}
```

### MySQL
``` go
import "github.com/edwingeng/wuid/mysql/wuid"

openDB := func() (*sql.DB, bool, error) {
    var db *sql.DB
    // ...
    return db, true, nil
}

// Setup
w := NewWUID("alpha", nil)
err := w.LoadH28FromMysql(openDB, "wuid")
if err != nil {
    panic(err)
}

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", w.Next())
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
w := NewWUID("alpha", nil)
err := w.LoadH28FromMongo(newClient, "test", "wuid", "default")
if err != nil {
    panic(err)
}

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", w.Next())
}
```

### Callback
``` go
import "github.com/edwingeng/wuid/callback/wuid"

callback := func() (int64, func(), error) {
    var h28 int64
    // ...
    return h28, nil, nil
}

// Setup
w := NewWUID("alpha", nil)
err := w.LoadH28WithCallback(callback)
if err != nil {
    panic(err)
}

// Generate
for i := 0; i < 10; i++ {
    fmt.Printf("%#016x\n", w.Next())
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

- `WithSection` brands a section ID on each generated number. A section ID must be in between [0, 7].
- `WithStep` sets the step and the floor for each generated number.
- `WithObfuscation` enables number obfuscation.

# Attentions
It is highly recommended to pass a logger to `wuid.NewWUID` and keep an eye on the warnings that include "renew failed". It indicates that the low 36 bits are about to run out in hours to hundreds of hours, and the renewal program failed for some reason. `WUID` will make many renewal attempts until succeeded. 

# Special thanks
- [dustinfog](https://github.com/dustinfog)

# Ports
- swift - https://github.com/ekscrypto/SwiftWUID
