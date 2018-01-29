package wuid

import (
	"errors"

	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis"
)

type Logger interface {
	internal.Logger
}

type WUID struct {
	w *internal.WUID
}

func NewWUID(tag string, logger Logger, opts ...Option) *WUID {
	var opts2 []internal.Option
	for _, opt := range opts {
		opts2 = append(opts2, internal.Option(opt))
	}
	return &WUID{w: internal.NewWUID(tag, logger, opts2...)}
}

func (this *WUID) Next() uint64 {
	return this.w.Next()
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
	if v == 0 {
		return errors.New("the h24 should not be 0")
	}

	this.w.Reset(uint64(v) << 40)

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24FromRedis(addr, pass, key)
	}

	return nil
}

type Option internal.Option

func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
