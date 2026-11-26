package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
	"github.com/makmanu/golang_todoapp/internal/core/postgres/pool"
)

type failingRows struct {
	closed bool
	err    error
}

func (r *failingRows) Scan(...any) error { return r.err }
func (r *failingRows) Next() bool        { return false }
func (r *failingRows) Close()            { r.closed = true }
func (r *failingRows) Err() error        { return r.err }

type fakeRow struct{ err error }

func (r fakeRow) Scan(...any) error { return r.err }

type fakeTag int64

func (t fakeTag) RowsAffected() int64 { return int64(t) }

type fakePool struct {
	ctx  context.Context
	rows *failingRows
	err  error
}

func (p *fakePool) Query(ctx context.Context, _ string, _ ...any) (pool.Rows, error) {
	p.ctx = ctx
	return p.rows, nil
}
func (p *fakePool) QueryRow(ctx context.Context, _ string, _ ...any) pool.Row {
	p.ctx = ctx
	return fakeRow{p.err}
}
func (p *fakePool) Exec(ctx context.Context, _ string, _ ...any) (pool.CommandTag, error) {
	p.ctx = ctx
	return fakeTag(0), p.err
}
func (p *fakePool) Close()                   {}
func (p *fakePool) OpTimeout() time.Duration { return time.Second }
func TestRowsErrorAndClose(t *testing.T) {
	sentinel := errors.New("stream interrupted")
	p := &fakePool{rows: &failingRows{err: sentinel}}
	_, err := NewUsersRepository(p).GetUsers(context.Background(), nil, nil)
	if !errors.Is(err, sentinel) || !p.rows.closed {
		t.Fatalf("err=%v closed=%v", err, p.rows.closed)
	}
	if _, ok := p.ctx.Deadline(); !ok {
		t.Fatal("query has no deadline")
	}
	if !errors.Is(p.ctx.Err(), context.Canceled) {
		t.Fatal("query context was not cancelled")
	}
}
func TestRepositoryMapsErrors(t *testing.T) {
	u, _ := domain.CreateUser("Name", nil)
	p := &fakePool{err: pool.ErrNoRows}
	r := NewUsersRepository(p)
	if _, err := r.GetUser(context.Background(), u.ID); !errors.Is(err, coreerrors.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := r.UpdateUser(context.Background(), u); !errors.Is(err, coreerrors.ErrConflict) {
		t.Fatal(err)
	}
	p.err = nil
	if err := r.DeleteUser(context.Background(), u.ID); !errors.Is(err, coreerrors.ErrNotFound) {
		t.Fatal(err)
	}
	p.err = pool.ErrViolatesForeignKey
	if err := r.DeleteUser(context.Background(), u.ID); !errors.Is(err, coreerrors.ErrConflict) {
		t.Fatal(err)
	}
}
