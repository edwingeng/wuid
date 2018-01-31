package wuid

import (
	"errors"
	"time"

	"github.com/edwingeng/wuid/internal"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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

func (this *WUID) LoadH24FromMongo(addr, user, pass, dbName, coll, docId string) error {
	return this.LoadH24FromMongoWithTimeout(addr, user, pass, dbName, coll, docId, 10*time.Second)
}

func (this *WUID) LoadH24FromMongoWithTimeout(addr, user, pass, dbName, coll, docId string, dialTimeout time.Duration) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty. tag: " + this.w.Tag)
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty. tag: " + this.w.Tag)
	}
	if len(coll) == 0 {
		return errors.New("coll cannot be empty. tag: " + this.w.Tag)
	}
	if len(docId) == 0 {
		return errors.New("docId cannot be empty. tag: " + this.w.Tag)
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
	_, err = c.FindId(docId).Apply(change, &m)
	if err != nil {
		return err
	}
	if err = this.w.VerifyH24(uint64(m["n"].(int))); err != nil {
		return err
	}

	this.w.Reset(uint64(m["n"].(int)) << 40)

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24FromMongo(addr, user, pass, dbName, coll, docId)
	}

	return nil
}

type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
