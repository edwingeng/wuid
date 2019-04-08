/*
Package wuid provides WUID, an extremely fast unique number generator. It is 10-135 times faster
than UUID and 4600 times faster than generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edwingeng/wuid/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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
func (this *WUID) Next() uint64 {
	return this.w.Next()
}

type NewClient func() (client *mongo.Client, autoDisconnect bool, err error)

// LoadH24FromMongoWithTimeout adds 1 to a specific number in your MongoDB, fetches its new value,
// and then sets that as the high 24 bits of the unique numbers that Next generates.
func (this *WUID) LoadH24FromMongo(newClient NewClient, dbName, coll, docID string) error {
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty. tag: " + this.w.Tag)
	}
	if len(coll) == 0 {
		return errors.New("coll cannot be empty. tag: " + this.w.Tag)
	}
	if len(docID) == 0 {
		return errors.New("docID cannot be empty. tag: " + this.w.Tag)
	}

	client, autoDisconnect, err := newClient()
	if err != nil {
		return err
	}
	if autoDisconnect {
		defer func() {
			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel2()
			_ = client.Disconnect(ctx2)
		}()
	}

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel1()
	if err := client.Ping(ctx1, readpref.Primary()); err != nil {
		return err
	}

	collOpts := &options.CollectionOptions{
		ReadConcern:    readconcern.Majority(),
		WriteConcern:   writeconcern.New(writeconcern.WMajority()),
		ReadPreference: readpref.Primary(),
	}
	c := client.Database(dbName).Collection(coll, collOpts)

	filter := bson.D{{"_id", docID}}
	update := bson.D{{"$inc", bson.D{{"n", int32(1)}}}}
	var findOneAndUpdateOptions options.FindOneAndUpdateOptions
	findOneAndUpdateOptions.SetUpsert(true).SetReturnDocument(options.After)
	var doc struct {
		N int32
	}
	err = c.FindOneAndUpdate(ctx1, filter, update, &findOneAndUpdateOptions).Decode(&doc)
	if err != nil {
		return err
	}
	h24 := uint64(doc.N)
	if err = this.w.VerifyH24(h24); err != nil {
		return err
	}

	this.w.Reset(h24 << 40)
	this.w.Logger.Info(fmt.Sprintf("<wuid> new h24: %d. tag: %s", h24, this.w.Tag))

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24FromMongo(newClient, dbName, coll, docID)
	}

	return nil
}

// RenewNow reacquires the high 24 bits from your data store immediately
func (this *WUID) RenewNow() error {
	return this.w.RenewNow()
}

// Option should never be used directly.
type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}

// WithH24Verifier sets your own h24 verifier
func WithH24Verifier(cb func(h24 uint64) error) Option {
	return Option(internal.WithH24Verifier(cb))
}
