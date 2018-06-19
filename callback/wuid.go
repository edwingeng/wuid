/*
Package wuid provides WUID, an extremely fast unique number generator. It is 10-135 times faster
than UUID and 4600 times faster than generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/edwingeng/wuid/internal"
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

// LoadH24WithCallback calls cb to get a number, and then sets it as the high 24 bits of the unique
// numbers that Next generates.
// The number returned by cb should look like 0x000123, not 0x0001230000000000.
func (this *WUID) LoadH24WithCallback(cb func() (uint64, error)) error {
	if cb == nil {
		return errors.New("cb cannot be nil. tag: " + this.w.Tag)
	}

	h24, err := cb()
	if err != nil {
		return err
	}

	if err = this.w.VerifyH24(h24); err != nil {
		return err
	}
	if this.w.Section == 0 {
		if h24 == atomic.LoadUint64(&this.w.N)>>40 {
			return fmt.Errorf("the h24 should be a different value other than %d. tag: %s", h24, this.w.Tag)
		}
	} else {
		if h24 == (atomic.LoadUint64(&this.w.N)>>40)&0x0FFFFF {
			return fmt.Errorf("the h20 should be a different value other than %d. tag: %s", h24, this.w.Tag)
		}
	}

	this.w.Reset(h24 << 40)
	this.w.Logger.Info(fmt.Sprintf("<wuid> new h24: %d. tag: %s", h24, this.w.Tag))

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24WithCallback(cb)
	}

	return nil
}

// RenewNow reacquires the high 24 bits from your data store immediately
func (this *WUID) RenewNow() error {
	return this.w.RenewNow()
}

// Option should never be used directly.
type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}

// WithH24Verifier sets your own h24 verifier
func WithH24Verifier(cb func(h24 uint64) error) Option {
	return Option(internal.WithH24Verifier(cb))
}
