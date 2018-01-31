package internal

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	CriticalValue uint64 = (1 << 40) * 8 / 10
	RenewInterval uint64 = 0x01FFFFFFFF
)

type WUID struct {
	sync.Mutex
	Section uint8
	N       uint64
	Tag     string
	Logger  Logger
	Renew   func() error
}

func NewWUID(tag string, logger Logger, opts ...Option) *WUID {
	w := &WUID{Tag: tag, Logger: logger}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

func (this *WUID) Next() uint64 {
	x := atomic.AddUint64(&this.N, 1)
	if x&0xFFFFFFFFFF >= CriticalValue && x&RenewInterval == 0 {
		this.Lock()
		renew := this.Renew
		this.Unlock()

		go func() {
			defer func() {
				if r := recover(); r != nil && this.Logger != nil {
					this.Logger.Warn(fmt.Sprintf("[wuid] panic. tag: %s, reason: %+v", this.Tag, r))
				}
			}()

			err := renew()
			if this.Logger == nil {
				return
			}
			if err != nil {
				this.Logger.Warn(fmt.Sprintf("[wuid] renew failed. tag: %s, reason: %s", this.Tag, err.Error()))
			} else {
				this.Logger.Info(fmt.Sprintf("[wuid] renew succeeded. tag: %s", this.Tag))
			}
		}()
	}
	return x
}

func (this *WUID) Reset(n uint64) {
	if this.Section == 0 {
		atomic.StoreUint64(&this.N, n)
	} else {
		atomic.StoreUint64(&this.N, n&0x0FFFFFFFFFFFFFFF|uint64(this.Section)<<60)
	}
}

func (this *WUID) VerifyH24(h24 uint64) error {
	if h24 == 0 {
		return errors.New("the h24 should not be 0")
	}

	if this.Section == 0 {
		if h24 > 0xFFFFFF {
			return errors.New("the h24 should not exceed 0xFFFFFF")
		}
	} else {
		if h24 > 0x0FFFFF {
			return errors.New("the h20 should not exceed 0x0FFFFF")
		}
	}

	return nil
}

type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
}

type Option func(*WUID)

func WithSection(section uint8) Option {
	if section == 0 || section >= 16 {
		panic("section must be in between [1, 15]")
	}
	return func(w *WUID) {
		w.Section = section
	}
}
