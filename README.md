# Overview
- WUID is a unique number generator, but not a UUID implementation.
- WUID is **100** times faster than UUID and **4600** times faster than generating unique numbers with Redis.
- WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store. By now, Redis, MySQL, and MongoDB are supported.

# Benchmarks
```
BenchmarkWUID           200000000            9.38 ns/op        0 B/op          0 allocs/op
BenchmarkRand           100000000           21.6  ns/op        0 B/op          0 allocs/op
BenchmarkTimestamp        2000000          669    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V1          2000000          888    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V2          2000000          904    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V4          1000000         1325    ns/op       16 B/op          1 allocs/op
BenchmarkRedis              30000        43970    ns/op      176 B/op          5 allocs/op
```

# Features
- Extremely fast
- Thread-safe
- Being unique within a data center
- Being unique across time
- Auto-renew when the low 40 bits are about to run out

# Install
``` bash
go get -u github.com/edwingeng/wuid
```

# Usage examples
### Redis
``` go
import "github.com/edwingeng/wuid/redis"

// Setup
g := wuid.NewWUID("default", nil)
g.LoadH24FromRedis("127.0.0.1:6379", "", "wuid")

// Generate
for i := 0; i < 10; i++ {
    fmt.Println(g.Next())
}
```

### MySQL
``` go
import "github.com/edwingeng/wuid/mysql"

// Setup
g := wuid.NewWUID("default", nil)
g.LoadH24FromMysql("127.0.0.1:3306", "root", "", "test", "wuid")

// Generate
for i := 0; i < 10; i++ {
    fmt.Println(g.Next())
}
```

### MongoDB
``` go
import "github.com/edwingeng/wuid/mongo"

// Setup
g := wuid.NewWUID("default", nil)
g.LoadH24FromMongo("127.0.0.1:27017", "", "", "test", "foo", "wuid")

// Generate
for i := 0; i < 10; i++ {
    fmt.Println(g.Next())
}
```

# Mysql table creation
``` sql
CREATE TABLE `wuid` (
    `h` int(10) NOT NULL AUTO_INCREMENT,
    `x` tinyint(4) NOT NULL DEFAULT '0',
    PRIMARY KEY (`x`),
    UNIQUE KEY `h` (`h`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
```

# Best practices
- Use different keys/tables/docs for different purposes.
