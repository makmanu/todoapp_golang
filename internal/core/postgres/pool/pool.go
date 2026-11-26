package pool

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNoRows             = errors.New("no rows")
	ErrViolatesForeignKey = errors.New("foreign key violation")
	ErrUniqueViolation    = errors.New("unique violation")
	ErrCheckViolation     = errors.New("check violation")
)

type Row interface{ Scan(...any) error }
type Rows interface {
	Row
	Next() bool
	Close()
	Err() error
}
type CommandTag interface{ RowsAffected() int64 }
type Pool interface {
	Query(context.Context, string, ...any) (Rows, error)
	QueryRow(context.Context, string, ...any) Row
	Exec(context.Context, string, ...any) (CommandTag, error)
	Close()
	OpTimeout() time.Duration
}
