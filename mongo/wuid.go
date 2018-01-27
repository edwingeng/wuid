package wuid

import (
	"errors"
	"sync/atomic"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type WUID struct {
	n uint64
}

func NewWUID() *WUID {
	return &WUID{}
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

	this.n = uint64(m["n"].(int)) << 40
	return nil
}

func (this *WUID) Next() uint64 {
	return atomic.AddUint64(&this.n, 1)
}
