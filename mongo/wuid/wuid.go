package wuid

import (
	"context"
	"errors"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
)

// WUID is an extremely fast universal unique identifier generator.
type WUID struct {
	w *internal.WUID
}

// NewWUID creates a new WUID instance.
func NewWUID(name string, logger slog.Logger, opts ...Option) *WUID {
	return &WUID{w: internal.NewWUID(name, logger, opts...)}
}

// Next returns a unique identifier.
func (w *WUID) Next() int64 {
	return w.w.Next()
}

type NewClient func() (client *mongo.Client, autoDisconnect bool, err error)

// LoadH28FromMongo adds 1 to a specific number in MongoDB and fetches its new value.
// The new value is used as the high 28 bits of all generated numbers. In addition, all the
// arguments passed in are saved for future renewal.
func (w *WUID) LoadH28FromMongo(newClient NewClient, dbName, coll, docID string) error {
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty")
	}
	if len(coll) == 0 {
		return errors.New("coll cannot be empty")
	}
	if len(docID) == 0 {
		return errors.New("docID cannot be empty")
	}

	client, autoDisconnect, err := newClient()
	if err != nil {
		return err
	}
	defer func() {
		if autoDisconnect {
			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel2()
			_ = client.Disconnect(ctx2)
		}
	}()

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

	var doc struct {
		N int32
	}

	filter := bson.D{
		{Key: "_id", Value: docID},
	}
	update := bson.D{
		{
			Key: "$inc",
			Value: bson.D{
				{Key: "n", Value: int32(1)},
			},
		},
	}

	var findOneAndUpdateOptions options.FindOneAndUpdateOptions
	findOneAndUpdateOptions.SetUpsert(true).SetReturnDocument(options.After)
	c := client.Database(dbName).Collection(coll, collOpts)
	err = c.FindOneAndUpdate(ctx1, filter, update, &findOneAndUpdateOptions).Decode(&doc)
	if err != nil {
		return err
	}
	h28 := int64(doc.N)
	if err = w.w.VerifyH28(h28); err != nil {
		return err
	}

	w.w.Reset(h28 << 36)
	w.w.Logger.Infof("<wuid> new h28: %d. name: %s", h28, w.w.Name)

	w.w.Lock()
	defer w.w.Unlock()

	if w.w.Renew != nil {
		return nil
	}
	w.w.Renew = func() error {
		return w.LoadH28FromMongo(newClient, dbName, coll, docID)
	}

	return nil
}

// RenewNow reacquires the high 28 bits immediately.
func (w *WUID) RenewNow() error {
	return w.w.RenewNow()
}

type Option = internal.Option

// WithH28Verifier adds an extra verifier for the high 28 bits.
func WithH28Verifier(cb func(h28 int64) error) Option {
	return internal.WithH28Verifier(cb)
}

// WithSection brands a section ID on each generated number. A section ID must be in between [0, 7].
func WithSection(section int8) Option {
	return internal.WithSection(section)
}

// WithStep sets the step and the floor for each generated number.
func WithStep(step int64, floor int64) Option {
	return internal.WithStep(step, floor)
}

// WithObfuscation enables number obfuscation.
func WithObfuscation(seed int) Option {
	return internal.WithObfuscation(seed)
}
