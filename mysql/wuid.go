package wuid

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/edwingeng/wuid/internal"
	_ "github.com/go-sql-driver/mysql"
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

func (this *WUID) LoadH24FromMysql(addr, user, pass, dbName, table string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty. tag: " + this.w.Tag)
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

	var dsn string
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@tcp(" + addr + ")/" + dbName

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(fmt.Sprintf("REPLACE INTO %s (x) VALUES (0)", table))
	if err != nil {
		return err
	}
	lastInsertedId, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if err = this.w.VerifyH24(uint64(lastInsertedId)); err != nil {
		return err
	}

	this.w.Reset(uint64(lastInsertedId) << 40)

	this.w.Lock()
	defer this.w.Unlock()

	if this.w.Renew != nil {
		return nil
	}
	this.w.Renew = func() error {
		return this.LoadH24FromMysql(addr, user, pass, dbName, table)
	}

	return nil
}

type Option internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [1, 15].
// It occupies the highest 4 bits of the numbers.
func WithSection(section uint8) Option {
	return Option(internal.WithSection(section))
}
