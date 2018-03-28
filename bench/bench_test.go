package bench

import (
	"math/rand"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/edwingeng/wuid/redis"
	"github.com/go-redis/redis"
	"github.com/satori/go.uuid"
)

type simpleLogger struct{}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func BenchmarkWUID(b *testing.B) {
	g := wuid.NewWUID("default", sl)
	err := g.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Next()
	}

}

func BenchmarkRand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rand.Int63()
	}
}

func BenchmarkTimestamp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.Now().Nanosecond()
	}
}

func BenchmarkUUID_V1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.NewV1()
	}
}

func BenchmarkUUID_V2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.NewV2(128)
	}
}

func BenchmarkUUID_V3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.NewV3(uuid.NamespaceDNS, "example.com")
	}
}

func BenchmarkUUID_V4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.NewV4()
	}
}

func BenchmarkUUID_V5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.NewV5(uuid.NamespaceDNS, "example.com")
	}
}

func BenchmarkRedis(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	defer client.Close()
	b.ResetTimer()

	key := "foo:id"
	for i := 0; i < b.N; i++ {
		_, err := client.Incr(key).Result()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSnowflake(b *testing.B) {
	node, err := snowflake.NewNode(1)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		node.Generate()
	}
}
