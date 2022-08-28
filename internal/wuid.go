package internal

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/edwingeng/slog"
)

const (
	// PanicValue indicates when Next starts to panic
	PanicValue int64 = ((1 << 36) * 98 / 100) & ^1023
	// CriticalValue indicates when to renew the high 28 bits
	CriticalValue int64 = ((1 << 36) * 80 / 100) & ^1023
	// RenewIntervalMask indicates how often renew is performed if it fails
	RenewIntervalMask int64 = 0x20000000 - 1
)

// WUID is for internal use only.
type WUID struct {
	N     int64
	Step  int64
	Floor int64

	slog.Logger
	Name        string
	Monolithic  bool
	Section     int8
	H28Verifier func(h28 int64) error

	sync.Mutex
	Renew func() error
}

// NewWUID is for internal use only.
func NewWUID(name string, logger slog.Logger, opts ...Option) (w *WUID) {
	w = &WUID{Step: 1, Name: name, Monolithic: true}
	if logger != nil {
		w.Logger = logger
	} else {
		w.Logger = slog.NewConsoleLogger()
	}
	for _, opt := range opts {
		opt(w)
	}
	return
}

// Next is for internal use only.
func (w *WUID) Next() int64 {
	x := atomic.AddInt64(&w.N, w.Step)
	v := x & 0x0FFFFFFFFF
	if v >= PanicValue {
		atomic.CompareAndSwapInt64(&w.N, x, x&(0x07FFFFFF<<36)|PanicValue)
		panic(fmt.Errorf("<wuid> the low 36 bits are about to run out. name: %s", w.Name))
	}
	if v >= CriticalValue && v&RenewIntervalMask == 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					w.Warnf("<wuid> panic, renew failed. name: %s, reason: %+v", w.Name, r)
				}
			}()

			err := w.RenewNow()
			if err != nil {
				w.Warnf("<wuid> renew failed. name: %s, reason: %+v", w.Name, err)
			} else {
				w.Infof("<wuid> renew succeeded. name: %s", w.Name)
			}
		}()
	}
	if w.Floor == 0 {
		return x
	} else {
		return x / w.Floor * w.Floor
	}
}

// RenewNow reacquires the high 28 bits from your data store immediately
func (w *WUID) RenewNow() error {
	w.Lock()
	f := w.Renew
	w.Unlock()
	return f()
}

// Reset is for internal use only.
func (w *WUID) Reset(n int64) {
	if n < 0 {
		panic(fmt.Errorf("n should never be negative. name: %s", w.Name))
	}
	if w.Monolithic {
		atomic.StoreInt64(&w.N, n)
	} else {
		atomic.StoreInt64(&w.N, n&0x0FFFFFFFFFFFFFFF|int64(w.Section)<<60)
	}
}

// VerifyH28 is for internal use only.
func (w *WUID) VerifyH28(h28 int64) error {
	if h28 <= 0 {
		return errors.New("h28 must be positive. name: " + w.Name)
	}

	if w.Monolithic {
		if h28 > 0x07FFFFFF {
			return errors.New("h28 should not exceed 0x07FFFFFF. name: " + w.Name)
		}
	} else {
		if h28 > 0x00FFFFFF {
			return errors.New("h28 should not exceed 0x00FFFFFF. name: " + w.Name)
		}
	}

	if w.Monolithic {
		if h28 == atomic.LoadInt64(&w.N)>>36 {
			return fmt.Errorf("h28 should be a different value other than %d. name: %s", h28, w.Name)
		}
	} else {
		if h28 == atomic.LoadInt64(&w.N)>>36&0x00FFFFFF {
			return fmt.Errorf("h28 should be a different value other than %d. name: %s", h28, w.Name)
		}
	}

	if w.H28Verifier != nil {
		if err := w.H28Verifier(h28); err != nil {
			return err
		}
	}

	return nil
}

// Option is for internal use only.
type Option func(*WUID)

// WithSection is for internal use only.
func WithSection(section int8) Option {
	if section < 0 || section > 7 {
		panic("section must be in between [0, 7]")
	}
	return func(w *WUID) {
		w.Monolithic = false
		w.Section = section
	}
}

// WithH28Verifier is for internal use only.
func WithH28Verifier(cb func(h28 int64) error) Option {
	return func(w *WUID) {
		w.H28Verifier = cb
	}
}

// WithStep sets the step and floor of Next()
func WithStep(step int64, floor int64) Option {
	switch step {
	case 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024:
	default:
		panic("the step must be one of these values: 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024")
	}
	if floor != 0 && (floor < 0 || floor >= step) {
		panic(fmt.Errorf("floor must be in between [0, %d)", step))
	}
	return func(wuid *WUID) {
		wuid.Step = step
		wuid.Floor = floor
	}
}
