package internal

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/edwingeng/slog"
)

const (
	// CriticalValue indicates when the low 36 bits are about to run out
	CriticalValue uint64 = (1 << 36) * 80 / 100
	// RenewInterval indicates how often renew retries are performed
	RenewInterval uint64 = 0x3FFFFFFF
	// PanicValue indicates when Next starts to panic
	PanicValue uint64 = (1 << 36) * 96 / 100
)

// WUID is for internal use only.
type WUID struct {
	sync.Mutex
	Section     uint8
	N           uint64
	Tag         string
	Logger      slog.Logger
	Renew       func() error
	H28Verifier func(h28 uint64) error
}

// NewWUID is for internal use only.
func NewWUID(tag string, logger slog.Logger, opts ...Option) *WUID {
	w := &WUID{Tag: tag}
	if logger != nil {
		w.Logger = logger
	} else {
		w.Logger = slog.NewConsoleLogger()
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Next is for internal use only.
func (this *WUID) Next() uint64 {
	x := atomic.AddUint64(&this.N, 1)
	v := x & 0xFFFFFFFFF
	if v >= PanicValue {
		atomic.StoreUint64(&this.N, 0xFFFFFFFFF)
		panic("<wuid> the low 36 bits are about to run out")
	}
	if v >= CriticalValue && v&RenewInterval == 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					this.Logger.Warnf("<wuid> panic, renew failed. tag: %s, reason: %+v", this.Tag, r)
				}
			}()

			err := this.RenewNow()
			if err != nil {
				this.Logger.Warnf("<wuid> renew failed. tag: %s, reason: %+v", this.Tag, err)
			} else {
				this.Logger.Infof("<wuid> renew succeeded. tag: %s", this.Tag)
			}
		}()
	}
	return x
}

// RenewNow reacquires the high 28 bits from your data store immediately
func (this *WUID) RenewNow() error {
	this.Lock()
	renew := this.Renew
	this.Unlock()

	return renew()
}

// Reset is for internal use only.
func (this *WUID) Reset(n uint64) {
	if this.Section == 0 {
		atomic.StoreUint64(&this.N, n)
	} else {
		atomic.StoreUint64(&this.N, n&0x0FFFFFFFFFFFFFFF|uint64(this.Section)<<60)
	}
}

// VerifyH28 is for internal use only.
func (this *WUID) VerifyH28(h28 uint64) error {
	if h28 == 0 {
		return errors.New("the h28 should not be 0. tag: " + this.Tag)
	}

	if this.Section == 0 {
		if h28 > 0x0FFFFFFF {
			return errors.New("the h28 should not exceed 0x0FFFFFFF. tag: " + this.Tag)
		}
	} else {
		if h28 > 0x00FFFFFF {
			return errors.New("the h28 should not exceed 0x00FFFFFF. tag: " + this.Tag)
		}
	}

	if this.H28Verifier != nil {
		if err := this.H28Verifier(h28); err != nil {
			return err
		}
	}

	return nil
}

// Option is for internal use only.
type Option func(*WUID)

// WithSection is for internal use only.
func WithSection(section uint8) Option {
	if section < 1 || section > 15 {
		panic("section must be in between [1, 15]")
	}
	return func(w *WUID) {
		w.Section = section
	}
}

// WithH28Verifier is for internal use only.
func WithH28Verifier(cb func(h28 uint64) error) Option {
	return func(w *WUID) {
		w.H28Verifier = cb
	}
}
