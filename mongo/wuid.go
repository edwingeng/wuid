package wuid

import (
	"errors"
	"sync/atomic"

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

func NewWUID(tag string, logger Logger) *WUID {
	return &WUID{w: internal.NewWUID(tag, logger)}
}

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

	atomic.StoreUint64(&this.w.N, uint64(m["n"].(int))<<40)

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
