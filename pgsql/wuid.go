/*
Package wuid provides WUID, an extremely fast unique number generator. It is 10-135 times faster
than UUID and 4600 times faster than generating unique numbers with Redis.

WUID generates unique 64-bit integers in sequence. The high 24 bits are loaded from a data store.
By now, Redis, MySQL, and MongoDB are supported.
*/
package wuid

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/edwingeng/wuid/internal"
	_ "github.com/lib/pq" // postgres driver
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

// LoadH24FromPgWithOpts adds 1 to a specific number in your PostgreSQL, fetches its new value, and then
// sets that as the high 24 bits of the unique numbers that Next generates.
// See https://godoc.org/github.com/lib/pq for valid options.
func (this *WUID) LoadH24FromPgWithOpts(host, user, pass, dbName, table, key, sslMode string, timeout int, sslCert, sslKey, sslrootcert string) error {
	if len(host) == 0 {
		return errors.New("host cannot be empty. tag: " + this.w.Tag)
	}
	if len(user) == 0 {
		return errors.New("user cannot be empty. tag: " + this.w.Tag)
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty. tag: " + this.w.Tag)
	}
	if len(table) == 0 {
		return errors.New("table cannot be empty. tag: " + this.w.Tag)
	}
	if len(sslMode) == 0 {
		return errors.New("sslMode cannot be empty. tag: " + this.w.Tag)
	}

	// Create connection string
	dsn := "postgres://"
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + fmt.Sprintf("'%s'", pass) // single quotes to handle whitespace in password
	}
	dsn += "@" + host
	dsn += "/" + dbName
	dsn += "?sslmode=" + sslMode
	dsn += "?connect_timeout=" + fmt.Sprintf("%v", timeout)
	if len(sslCert) > 0 {
		dsn += "?sslcert=" + fmt.Sprintf("'%s'", sslCert) // single quotes to handle whitespace
	}
	if len(sslKey) > 0 {
		dsn += "?sslkey=" + fmt.Sprintf("'%s'", sslKey) // single quotes to handle whitespace
	}
	if len(sslrootcert) > 0 {
		dsn += "?sslrootcert=" + fmt.Sprintf("'%s'", sslrootcert) // single quotes to handle whitespace
	}

	return this.loadH24FromPg(dsn, table)
}

// LoadH24FromPg adds 1 to a specific number in your PostgreSQL, fetches its new value, and then
// sets that as the high 24 bits of the unique numbers that Next generates.
func (this *WUID) LoadH24FromPg(host, user, pass, dbName, table string) error {
	if len(host) == 0 {
		return errors.New("host cannot be empty. tag: " + this.w.Tag)
	}
	if len(user) == 0 {
		return errors.New("user cannot be empty. tag: " + this.w.Tag)
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty. tag: " + this.w.Tag)
	}
	if len(table) == 0 {
		return errors.New("table cannot be empty. tag: " + this.w.Tag)
	}

	// Create connection string
	dsn := "postgres://"
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@" + host
	dsn += "/" + dbName

	return this.loadH24FromPg(dsn, table)
}

// loadH24FromPg adds 1 to a specific number in your PostgreSQL, fetches its new value, and then
// sets that as the high 24 bits of the unique numbers that Next generates.
func (this *WUID) loadH24FromPg(dsn, table string) error {

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("db connection error: %s , with connection: %s", err, dsn)
	}
	defer db.Close()

	/*
		result, err := db.Exec(fmt.Sprintf("REPLACE INTO %s (x) VALUES (0)", table))
		if err != nil {
			return err
		}
	*/

	var lastInsertedID int64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (x) VALUES (0) ON CONFLICT (x) DO UPDATE SET h = %s.h + 1 returning h", table, table)).Scan(&lastInsertedID)
	if err != nil {
		return err
	}
	fmt.Println(lastInsertedID)
	/*
		lastInsertedID, err := result.LastInsertId()
		if err != nil {
			return err
		}
	*/

	h24 := uint64(lastInsertedID)
	if err = this.w.VerifyH24(h24); err != nil {
		return err
	}

	this.w.Reset(h24 << 40)
	this.w.Logger.Info(fmt.Sprintf("<wuid> new h24: %d", h24))

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.loadH24FromPg(dsn, table)
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