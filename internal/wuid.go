package internal

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	criticalValue uint64 = (1 << 40) - 86400*1000000
)

type WUID struct {
	sync.Mutex
	N      uint64
	Tag    string
	Logger Logger
	Renew  func() error
}

func NewWUID(tag string, logger Logger) *WUID {
	return &WUID{Tag: tag, Logger: logger}
}

func (this *WUID) Next() uint64 {
	x := atomic.AddUint64(&this.N, 1)
	if x&0xFFFFFFFFFF >= criticalValue && x&0x01FFFFFF == 0 {
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

type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
}
