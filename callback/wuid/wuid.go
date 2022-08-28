package wuid

import (
	"errors"

	"github.com/edwingeng/slog"
	"github.com/edwingeng/wuid/internal"
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

type H28Callback func() (h28 int64, clean func(), err error)

// LoadH28WithCallback invokes cb to get a number, and then sets it as the high 28 bits of
// the unique numbers that Next generates.
func (w *WUID) LoadH28WithCallback(cb H28Callback) error {
	if cb == nil {
		return errors.New("cb cannot be nil. name: " + w.w.Name)
	}

	h28, clean, err := cb()
	if err != nil {
		return err
	} else if clean != nil {
		defer clean()
	}

	if err = w.w.VerifyH28(h28); err != nil {
		return err
	}

	w.w.Reset(h28 << 36)
	w.w.Infof("<wuid> new h28: %d. name: %s", h28, w.w.Name)

	w.w.Lock()
	defer w.w.Unlock()

	if w.w.Renew != nil {
		return nil
	}
	w.w.Renew = func() error {
		return w.LoadH28WithCallback(cb)
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
