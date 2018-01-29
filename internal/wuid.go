package internal

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	criticalValue uint64 = (1 << 40) * 8 / 10
	renewInterval uint64 = 0x01FFFFFFFF
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
	if x&0xFFFFFFFFFF >= criticalValue && x&renewInterval == 0 {
		this.Lock()
		renew := this.Renew
		this.Unlock()

		go func() {
			err := renew()
			if this.Logger == nil {
				return
			}
			if err != nil {
				this.Logger.Warn(fmt.Sprintf("renew failed. tag: %s, reason: %s", this.Tag, err.Error()))
			} else {
				this.Logger.Info(fmt.Sprintf("renew succeeded. tag: %s", this.Tag))
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
