/*
WUID is a unique number generator. It is 10-135 times faster than UUID and 4600 times faster than
generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"errors"

	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis"
)

/*
Logger includes internal.Logger, while internal.Logger includes:
	Info(args ...interface{})
	Warn(args ...interface{})
*/
type Logger interface {
	internal.Logger
}

// WUID is an extremely fast unique number generator.
type WUID struct {
	w *internal.WUID
}

// NewWUID creates a new WUID instance.
func NewWUID(tag string, logger Logger, opts ...Option) *WUID {
	var opts2 []internal.Option
	for _, opt := range opts {
		opts2 = append(opts2, internal.Option(opt))
	}
	return &WUID{w: internal.NewWUID(tag, logger, opts2...)}
}

// Next returns the next unique number.
func (this *WUID) Next() uint64 {
	return this.w.Next()
}

// LoadH24FromRedis adds 1 to a specific number in your Redis, fetches the new value, and then
// sets it as the high 24 bits of the unique numbers that Next generates.
func (this *WUID) LoadH24FromRedis(addr, pass, key string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty. tag: " + this.w.Tag)
	}
	if len(key) == 0 {
		return errors.New("key cannot be empty. tag: " + this.w.Tag)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer client.Close()

	h24, err := client.Incr(key).Result()
	if err != nil {
		return err
	}
	if err = this.w.VerifyH24(uint64(h24)); err != nil {
		return err
	}

	this.w.Reset(uint64(h24) << 40)

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

// You should never use Option directly.
type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
