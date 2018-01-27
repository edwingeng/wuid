package wuid

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis"
)

const (
	criticalValue uint64 = (1 << 40) - 86400*1000000
)

type WUID struct {
	sync.Mutex
	n      uint64
	tag    string
	logger *log.Logger
	renew  func() error
}

func NewWUID(tag string, logger *log.Logger) *WUID {
	return &WUID{tag: tag, logger: logger}
}

func (this *WUID) LoadH24FromRedis(addr, pass, key string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	if len(key) == 0 {
		return errors.New("key cannot be empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	defer client.Close()

	v, err := client.Incr(key).Result()
	if err != nil {
		return err
	}

	atomic.StoreUint64(&this.n, uint64(v)<<40)

	this.Lock()
	defer this.Unlock()

	if this.renew != nil {
		return nil
	}
	this.renew = func() error {
		return this.LoadH24FromRedis(addr, pass, key)
	}

	return nil
}

func (this *WUID) Next() uint64 {
	x := atomic.AddUint64(&this.n, 1)
	if x&0xFFFFFFFFFF >= criticalValue && x&0x01FFFFFF == 0 {
		this.Lock()
		renew := this.renew
		this.Unlock()

		go func() {
			err := renew()
			if this.logger == nil {
				return
			}
			if err != nil {
				this.logger.Println(fmt.Sprintf("renew failed. tag: %s, reason: %s", this.tag, err.Error()))
			} else {
				this.logger.Println(fmt.Sprintf("renew succeeded. tag: %s", this.tag))
			}
		}()
	}
	return x
}
