# Overview
WUID is NOT a UUID implementation. It generates 'unique' 64-bit integers in sequence. The high 24 bits are loaded from a data source. By now, Redis, MySQL, and MongoDB are supported.

# Benchmarks
```
BenchmarkWUID           200000000            9.38 ns/op        0 B/op          0 allocs/op
BenchmarkWUID-4         200000000            9.19 ns/op        0 B/op          0 allocs/op
BenchmarkRand           100000000           21.6  ns/op        0 B/op          0 allocs/op
BenchmarkRand-4         100000000           21.9  ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V1          2000000          888    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V1-4        2000000          871    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V2          2000000          904    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V2-4        2000000          887    ns/op        0 B/op          0 allocs/op
BenchmarkUUID_V4          1000000         1325    ns/op       16 B/op          1 allocs/op
BenchmarkUUID_V4-4        1000000         1287    ns/op       16 B/op          1 allocs/op
BenchmarkRedis              30000        43970    ns/op      176 B/op          5 allocs/op
BenchmarkRedis-4            30000        42279    ns/op      176 B/op          5 allocs/op
```

# Features
- Extremely fast
- Thread-safe
- Being unique within a data center
- Being unique across time
- Auto renew when the lower 40 bits are about to run out

# Usage examples
## Redis
``` go
import "github.com/edwingeng/wuid/redis"

g := wuid.NewWUID("default", nil)
g.LoadH24FromRedis("127.0.0.1:6379", "", "wuid")
fmt.Println(g.Next())
```

## MySQL
``` go
import "github.com/edwingeng/wuid/mysql"

g := wuid.NewWUID("default", nil)
g.LoadH24FromMysql("127.0.0.1:3306", "root", "", "test", "wuid")
fmt.Println(g.Next())
```

## MongoDB
``` go
import "github.com/edwingeng/wuid/mongo"

g := wuid.NewWUID("default", nil)
g.LoadH24FromMongo("127.0.0.1:27017", "", "", "test", "foo", "wuid")
fmt.Println(g.Next())
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
