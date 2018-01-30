package wuid

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/edwingeng/wuid/internal"
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

// Next returns the next unique number.
func (this *WUID) Next() uint64 {
	return this.w.Next()
}

// The return value should look like 0x000123, not 0x0001230000000000.
func (this *WUID) LoadH24WithCallback(cb func() (uint64, error)) error {
	if cb == nil {
		return errors.New("cb cannot be nil")
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
			return errors.New(fmt.Sprintf("the h24 should be a different value other than %d", h24))
		}
	} else {
		if h24 == (atomic.LoadUint64(&this.w.N)>>40)&0x0FFFFF {
			return errors.New(fmt.Sprintf("the h20 should be a different value other than %d", h24))
		}
	}

	this.w.Reset(uint64(h24) << 40)

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

type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
