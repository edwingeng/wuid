/*
Package wuid provides WUID, an extremely fast unique number generator. It is 10-135 times faster
than UUID and 4600 times faster than generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"errors"
	"time"

	"github.com/edwingeng/wuid/internal"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
func (ego *WUID) Next() uint64 {
	return ego.w.Next()
}

// LoadH24FromMongo adds 1 to a specific number in your MongoDB, fetches the new value,
// and then sets it as the high 24 bits of the unique numbers that Next generates.
func (ego *WUID) LoadH24FromMongo(addr, user, pass, dbName, coll, docID string) error {
	return ego.LoadH24FromMongoWithTimeout(addr, user, pass, dbName, coll, docID, 3*time.Second)
}

// LoadH24FromMongoWithTimeout adds 1 to a specific number in your MongoDB, fetches the new value,
// and then sets it as the high 24 bits of the unique numbers that Next generates.
func (ego *WUID) LoadH24FromMongoWithTimeout(addr, user, pass, dbName, coll, docID string, dialTimeout time.Duration) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty. tag: " + ego.w.Tag)
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty. tag: " + ego.w.Tag)
	}
	if len(coll) == 0 {
		return errors.New("coll cannot be empty. tag: " + ego.w.Tag)
	}
	if len(docID) == 0 {
		return errors.New("docID cannot be empty. tag: " + ego.w.Tag)
	}

	var url = "mongodb://" + addr + "/" + coll
	mongo, err := mgo.DialWithTimeout(url, dialTimeout)
	if err != nil {
		return err
	}
	defer mongo.Close()

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"n": int32(1)}},
		Upsert:    true,
		ReturnNew: true,
	}
	if len(user) > 0 {
		if err = mongo.DB(dbName).Login(user, pass); err != nil {
			return err
		}
	}
	c := mongo.DB(dbName).C(coll)
	m := make(map[string]interface{})
	_, err = c.FindId(docID).Apply(change, &m)
	if err != nil {
		return err
	}
	if err = ego.w.VerifyH24(uint64(m["n"].(int))); err != nil {
		return err
	}

	ego.w.Reset(uint64(m["n"].(int)) << 40)

	ego.w.Lock()
	defer ego.w.Unlock()

	if ego.w.Renew != nil {
		return nil
	}
	ego.w.Renew = func() error {
		return ego.LoadH24FromMongo(addr, user, pass, dbName, coll, docID)
	}

	return nil
}

// RenewNow reacquires the high 24 bits from your data store immediately
func (ego *WUID) RenewNow() error {
	return ego.w.RenewNow()
}

// Option should never be used directly.
type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
