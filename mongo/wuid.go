package wuid

import (
	"errors"

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
	if len(addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty")
	}
	if len(coll) == 0 {
		return errors.New("coll cannot be empty")
	}
	if len(docId) == 0 {
		return errors.New("docId cannot be empty")
	}

	var url = "mongodb://"
	if len(user) > 0 {
		url += user
		if len(pass) > 0 {
			url += ":" + pass
		}
		url += "@"
	}
	url += addr
	url += "/" + coll

	mongo, err := mgo.Dial(url)
	if err != nil {
		return err
	}
	defer mongo.Close()

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"n": int32(1)}},
		Upsert:    true,
		ReturnNew: true,
	}
	c := mongo.DB(dbName).C(coll)
	m := make(map[string]interface{})
	_, err = c.FindId(docId).Apply(change, &m)
	if err != nil {
		return err
	}
	if m["n"].(int) == 0 {
		return errors.New("the h24 should not be 0")
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
