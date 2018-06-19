package internal

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

const (
	// CriticalValue indicates when the low 40 bits are about to run out
	CriticalValue uint64 = (1 << 40) * 80 / 100
	// RenewInterval indicates how often renew retries are performed
	RenewInterval uint64 = 0x01FFFFFFFF
	// PanicValue indicates when Next starts to panic
	PanicValue uint64 = (1 << 40) * 96 / 100
)

// WUID is for internal use only.
type WUID struct {
	sync.Mutex
	Section      uint8
	N            uint64
	Tag          string
	Logger       Logger
	Renew        func() error
	H24Validator func(h24 uint64) error
}

// NewWUID is for internal use only.
func NewWUID(tag string, logger Logger, opts ...Option) *WUID {
	w := &WUID{Tag: tag}
	if logger != nil {
		w.Logger = logger
	} else {
		w.Logger = defaultLogger{}
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Next is for internal use only.
func (this *WUID) Next() uint64 {
	x := atomic.AddUint64(&this.N, 1)
	if x&0xFFFFFFFFFF >= PanicValue {
		panic("<wuid> the low 40 bits are about to run out")
	}
	if x&0xFFFFFFFFFF >= CriticalValue && x&RenewInterval == 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					this.Logger.Warn(fmt.Sprintf("<wuid> panic, renew failed. tag: %s, reason: %+v", this.Tag, r))
				}
			}()

			err := this.RenewNow()
			if err != nil {
				this.Logger.Warn(fmt.Sprintf("<wuid> renew failed. reason: %s", err.Error()))
			} else {
				this.Logger.Info(fmt.Sprintf("<wuid> renew succeeded. tag: %s", this.Tag))
			}
		}()
	}
	return x
}

// RenewNow reacquires the high 24 bits from your data store immediately
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

// VerifyH24 is for internal use only.
func (this *WUID) VerifyH24(h24 uint64) error {
	if h24 == 0 {
		return errors.New("the h24 should not be 0. tag: " + this.Tag)
	}

	if this.Section == 0 {
		if h24 > 0xFFFFFF {
			return errors.New("the h24 should not exceed 0xFFFFFF. tag: " + this.Tag)
		}
	} else {
		if h24 > 0x0FFFFF {
			return errors.New("the h20 should not exceed 0x0FFFFF. tag: " + this.Tag)
		}
	}

	if this.H24Validator != nil {
		if err := this.H24Validator(h24); err != nil {
			return err
		}
	}

	return nil
}

// Logger is for internal use only.
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
}

type defaultLogger struct{}

func (l defaultLogger) Info(args ...interface{}) {
	log.Println(args...)
}

func (l defaultLogger) Warn(args ...interface{}) {
	log.Println(args...)
}

// Option is for internal use only.
type Option func(*WUID)

// WithSection is for internal use only.
func WithSection(section uint8) Option {
	if section == 0 || section >= 16 {
		panic("section must be in between [1, 15]")
	}
	return func(w *WUID) {
		w.Section = section
	}
}

// WithH24Validator is for internal use only.
func WithH24Validator(cb func(h24 uint64) error) Option {
	return func(w *WUID) {
		w.H24Validator = cb
	}
}
