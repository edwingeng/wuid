/*
Package wuid provides WUID, an extremely fast unique number generator. It is 10-135 times faster
than UUID and 4600 times faster than generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 28 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"errors"
	"io"

	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis"
)

// WUID is an extremely fast unique number generator.
type WUID struct {
	w *internal.WUID
}

// NewWUID creates a new WUID instance.
func NewWUID(tag string, logger slog.Logger, opts ...Option) *WUID {
	return &WUID{w: internal.NewWUID(tag, logger, opts...)}
}

// Next returns the next unique number.
func (this *WUID) Next() int64 {
	return this.w.Next()
}

type NewClient func() (client redis.Cmdable, autoDisconnect bool, err error)

// LoadH28FromRedis adds 1 to a specific number in your Redis, fetches its new value, and then
// sets that as the high 28 bits of the unique numbers that Next generates.
func (this *WUID) LoadH28FromRedis(newClient NewClient, key string) error {
	if len(key) == 0 {
		return errors.New("key cannot be empty. tag: " + this.w.Tag)
	}

	client, autoDisconnect, err := newClient()
	if err != nil {
		return err
	}
	if autoDisconnect {
		defer func() {
			closer := client.(io.Closer)
			_ = closer.Close()
		}()
	}

	n, err := client.Incr(key).Result()
	if err != nil {
		return err
	}
	h28 := int64(n)
	if err = this.w.VerifyH28(h28); err != nil {
		return err
	}

	this.w.Reset(h28 << 36)
	this.w.Logger.Infof("<wuid> new h28: %d. tag: %s", h28, this.w.Tag)

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH28FromRedis(newClient, key)
	}

	return nil
}

// RenewNow reacquires the high 28 bits from your data store immediately
func (this *WUID) RenewNow() error {
	return this.w.RenewNow()
}

type Option = internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 7].
// It occupies the highest 4 bits of the numbers.
func WithSection(section int8) Option {
	return internal.WithSection(section)
}

// WithH28Verifier sets your own h28 verifier
func WithH28Verifier(cb func(h28 int64) error) Option {
	return internal.WithH28Verifier(cb)
}
