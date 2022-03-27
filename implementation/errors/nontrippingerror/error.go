package aurestnontripping

import "context"

type NonTrippingError interface {
	Ctx() context.Context
	// IsNonTrippingError is a marker method.
	// The presence of this method makes the interface unique and thus recognizable by a simple type check.
	IsNonTrippingError() bool
}

type Impl struct {
	ctx context.Context
	err error
}

func New(ctx context.Context, err error) error {
	return &Impl{
		ctx: ctx,
		err: err,
	}
}

// implement error interface

func (e *Impl) Error() string {
	return e.err.Error()
}

// implement NonTrippingError

func (e *Impl) Ctx() context.Context {
	return e.ctx
}

func (e *Impl) IsNonTrippingError() bool {
	return true
}

// check for NonTrippingError

func Is(err error) bool {
	_, ok := err.(NonTrippingError)
	return ok
}
