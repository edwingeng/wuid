package islet

import (
	crypto_rand "crypto/rand"
	"math/rand"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/redis/wuid"
	"github.com/go-redis/redis"
	"github.com/oklog/ulid"
	"github.com/satori/go.uuid"
)

var vault struct {
	x1 int64
	x2 int
	x3 uuid.UUID
	x4 snowflake.ID
	x5 ulid.ULID
}

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func BenchmarkWUID(b *testing.B) {
	b.ReportAllocs()
	addr, pass, key := getRedisConfig()
	newClient := func() (client redis.Cmdable, autoDisconnect bool, err error) {
		return redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
		}), true, nil
	}

	g := wuid.NewWUID("default", slog.NewDumbLogger())
	err := g.LoadH28FromRedis(newClient, key)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vault.x1 = g.Next()
	}

}

func BenchmarkRand(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x1 = rand.Int63()
	}
}

func BenchmarkTimestamp(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x2 = time.Now().Nanosecond()
	}
}

func BenchmarkUUID_V1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x3 = uuid.NewV1()
	}
}

func BenchmarkUUID_V2(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x3 = uuid.NewV2(128)
	}
}

func BenchmarkUUID_V3(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x3 = uuid.NewV3(uuid.NamespaceDNS, "example.com")
	}
}

func BenchmarkUUID_V4(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x3 = uuid.NewV4()
	}
}

func BenchmarkUUID_V5(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		vault.x3 = uuid.NewV5(uuid.NamespaceDNS, "example.com")
	}
}

func BenchmarkRedis(b *testing.B) {
	b.ReportAllocs()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if err := client.Ping().Err(); err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = client.Close()
	}()
	b.ResetTimer()

	key := "foo:id"
	var err error
	for i := 0; i < b.N; i++ {
		vault.x1, err = client.Incr(key).Result()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSnowflake(b *testing.B) {
	b.ReportAllocs()
	node, err := snowflake.NewNode(1)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vault.x4 = node.Generate()
	}
}

func BenchmarkULID(b *testing.B) {
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		now := uint64(time.Now().UnixNano() / int64(time.Millisecond))
		vault.x5, err = ulid.New(now, crypto_rand.Reader)
		if err != nil {
			panic(err)
		}
	}
}
