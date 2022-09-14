package wuid

import (
	"errors"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis"
)

// WUID is an extremely fast universal unique identifier generator.
type WUID struct {
	w *internal.WUID
}

// NewWUID creates a new WUID instance.
func NewWUID(name string, logger slog.Logger, opts ...Option) *WUID {
	return &WUID{w: internal.NewWUID(name, logger, opts...)}
}

// Next returns a unique identifier.
func (w *WUID) Next() int64 {
	return w.w.Next()
}

type NewClient func() (client redis.UniversalClient, autoClose bool, err error)

// LoadH28FromRedis adds 1 to a specific number in Redis and fetches its new value.
// The new value is used as the high 28 bits of all generated numbers. In addition, all the
// arguments passed in are saved for future renewal.
func (w *WUID) LoadH28FromRedis(newClient NewClient, key string) error {
	if len(key) == 0 {
		return errors.New("key cannot be empty")
	}

	client, autoClose, err := newClient()
	if err != nil {
		return err
	}
	defer func() {
		if autoClose {
			_ = client.Close()
		}
	}()

	h28, err := client.Incr(key).Result()
	if err != nil {
		return err
	}
	if err = w.w.VerifyH28(h28); err != nil {
		return err
	}

	w.w.Reset(h28 << 36)
	w.w.Logger.Infof("<wuid> new h28: %d. name: %s", h28, w.w.Name)

	w.w.Lock()
	defer w.w.Unlock()

	if w.w.Renew != nil {
		return nil
	}
	w.w.Renew = func() error {
		return w.LoadH28FromRedis(newClient, key)
	}

	return nil
}

// RenewNow reacquires the high 28 bits immediately.
func (w *WUID) RenewNow() error {
	return w.w.RenewNow()
}

type Option = internal.Option

// WithH28Verifier adds an extra verifier for the high 28 bits.
func WithH28Verifier(cb func(h28 int64) error) Option {
	return internal.WithH28Verifier(cb)
}

// WithSection brands a section ID on each generated number. A section ID must be in between [0, 7].
func WithSection(section int8) Option {
	return internal.WithSection(section)
}

// WithStep sets the step and the floor for each generated number.
func WithStep(step int64, floor int64) Option {
	return internal.WithStep(step, floor)
}

// WithObfuscation enables number obfuscation.
func WithObfuscation(seed int) Option {
	return internal.WithObfuscation(seed)
}
