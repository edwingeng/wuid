package internal

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	// CriticalValue indicates when the low 40 bits are about to run out
	CriticalValue uint64 = (1 << 40) * 80 / 100
	// RenewInterval indicates how often renew retries are performed
	RenewInterval uint64 = 0x01FFFFFFFF
	// DangerLine indicates when Next starts to panic
	DangerLine uint64 = (1 << 40) * 96 / 100
)

// WUID is for internal use only.
type WUID struct {
	sync.Mutex
	Section uint8
	N       uint64
	Tag     string
	Logger  Logger
	Renew   func() error
}

// NewWUID is for internal use only.
func NewWUID(tag string, logger Logger, opts ...Option) *WUID {
	w := &WUID{Tag: tag, Logger: logger}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Next is for internal use only.
func (ego *WUID) Next() uint64 {
	x := atomic.AddUint64(&ego.N, 1)
	if x&0xFFFFFFFFFF >= DangerLine {
		panic(errors.New("[wuid] the low 40 bits are about to run out"))
	}
	if x&0xFFFFFFFFFF >= CriticalValue && x&RenewInterval == 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil && ego.Logger != nil {
					ego.Logger.Warn(fmt.Sprintf("[wuid] panic, renew failed. tag: %s, reason: %+v", ego.Tag, r))
				}
			}()

			err := ego.RenewNow()

			if ego.Logger == nil {
				return
			}
			if err != nil {
				ego.Logger.Warn(fmt.Sprintf("[wuid] renew failed. tag: %s, reason: %s", ego.Tag, err.Error()))
			} else {
				ego.Logger.Info(fmt.Sprintf("[wuid] renew succeeded. tag: %s", ego.Tag))
			}
		}()
	}
	return x
}

// RenewNow reacquires the high 24 bits from your data store immediately
func (ego *WUID) RenewNow() error {
	ego.Lock()
	renew := ego.Renew
	ego.Unlock()

	return renew()
}

// Reset is for internal use only.
func (ego *WUID) Reset(n uint64) {
	if ego.Section == 0 {
		atomic.StoreUint64(&ego.N, n)
	} else {
		atomic.StoreUint64(&ego.N, n&0x0FFFFFFFFFFFFFFF|uint64(ego.Section)<<60)
	}
}

// VerifyH24 is for internal use only.
func (ego *WUID) VerifyH24(h24 uint64) error {
	if h24 == 0 {
		return errors.New("the h24 should not be 0. tag: " + ego.Tag)
	}

	if ego.Section == 0 {
		if h24 > 0xFFFFFF {
			return errors.New("the h24 should not exceed 0xFFFFFF. tag: " + ego.Tag)
		}
	} else {
		if h24 > 0x0FFFFF {
			return errors.New("the h20 should not exceed 0x0FFFFF. tag: " + ego.Tag)
		}
	}

	return nil
}

// Logger is for internal use only.
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
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
