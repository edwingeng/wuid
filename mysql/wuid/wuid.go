package wuid

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
	_ "github.com/go-sql-driver/mysql" //...
)

// WUID is an extremely fast unique number generator.
type WUID struct {
	w *internal.WUID
}

// NewWUID creates a new WUID instance.
func NewWUID(name string, logger slog.Logger, opts ...Option) *WUID {
	return &WUID{w: internal.NewWUID(name, logger, opts...)}
}

// Next returns the next unique number.
func (w *WUID) Next() int64 {
	return w.w.Next()
}

type NewDB func() (client *sql.DB, autoDisconnect bool, err error)

// LoadH28FromMysql adds 1 to a specific number in your MySQL, fetches its new value, and then
// sets that as the high 28 bits of the unique numbers that Next generates.
func (w *WUID) LoadH28FromMysql(newDB NewDB, table string) error {
	if len(table) == 0 {
		return errors.New("table cannot be empty. name: " + w.w.Name)
	}

	db, autoDisconnect, err := newDB()
	if err != nil {
		return err
	}
	if autoDisconnect {
		defer func() {
			_ = db.Close()
		}()
	}

	result, err := db.Exec(fmt.Sprintf("REPLACE INTO %s (x) VALUES (0)", table))
	if err != nil {
		return err
	}
	h28, err := result.LastInsertId()
	if err != nil {
		return err
	}
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
		return w.LoadH28FromMysql(newDB, table)
	}

	return nil
}

// RenewNow reacquires the high 28 bits from your data store immediately
func (w *WUID) RenewNow() error {
	return w.w.RenewNow()
}

type Option = internal.Option

// WithSection adds a section ID to the generated numbers. The section ID must be in between [0, 7].
func WithSection(section int8) Option {
	return internal.WithSection(section)
}

// WithH28Verifier sets your own h28 verifier
func WithH28Verifier(cb func(h28 int64) error) Option {
	return internal.WithH28Verifier(cb)
}

// WithStep sets the step and floor of Next()
func WithStep(step int64, floor int64) Option {
	return internal.WithStep(step, floor)
}
