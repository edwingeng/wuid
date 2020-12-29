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

	h28, err := client.Incr(key).Result()
	if err != nil {
		return err
	}
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
func WithSection(section int8) Option {
	return internal.WithSection(section)
}

// WithH28Verifier sets your own h28 verifier
func WithH28Verifier(cb func(h28 int64) error) Option {
	return internal.WithH28Verifier(cb)
}

// WithStep sets the step and floor of Next()
func WithStep(step int64, floor int64) Option {
	return internal.WithStep(step, floor)
}
