package wuid

import (
	"errors"
	"sync/atomic"

	"github.com/go-redis/redis"
)

type WUID struct {
	n uint64
}

func NewWUID() *WUID {
	return &WUID{}
}

func (this *WUID) LoadH24FromRedis(addr, pass, key string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	if len(key) == 0 {
		return errors.New("key cannot be empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer client.Close()

	v, err := client.Incr(key).Result()
	if err != nil {
		return err
	}

	this.n = uint64(v&0x0FFF) << 40
	return nil
}

func (this *WUID) Next() uint64 {
	return atomic.AddUint64(&this.n, 1)
}
