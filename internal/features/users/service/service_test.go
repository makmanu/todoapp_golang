package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
)

type fakeRepository struct {
	user                  domain.User
	err                   error
	saves, updates, reads int
}

func (f *fakeRepository) SaveUser(_ context.Context, u domain.User) (domain.User, error) {
	f.saves++
	return u, f.err
}
func (f *fakeRepository) GetUser(context.Context, domain.UserID) (domain.User, error) {
	f.reads++
	return f.user, nil
}
func (f *fakeRepository) GetUsers(context.Context, *int, *int) ([]domain.User, error) {
	f.reads++
	return []domain.User{}, f.err
}
func (f *fakeRepository) UpdateUser(_ context.Context, u domain.User) (domain.User, error) {
	f.updates++
	u.Version++
	return u, f.err
}
func (f *fakeRepository) DeleteUser(context.Context, domain.UserID) error { return f.err }
func TestServiceRejectsInvalidDataBeforeWrite(t *testing.T) {
	f := &fakeRepository{}
	s := NewUsersService(f)
	if _, err := s.CreateUser(context.Background(), "ab", nil); !errors.Is(err, coreerrors.ErrInvalidArgument) || f.saves != 0 {
		t.Fatalf("%v, saves=%d", err, f.saves)
	}
	n := -1
	if _, err := s.GetUsers(context.Background(), &n, nil); !errors.Is(err, coreerrors.ErrInvalidArgument) || f.reads != 0 {
		t.Fatalf("%v", err)
	}
	if _, err := s.GetUsers(context.Background(), nil, &n); !errors.Is(err, coreerrors.ErrInvalidArgument) || f.reads != 0 {
		t.Fatalf("%v", err)
	}
}
func TestPatchConflictSurvivesWrapping(t *testing.T) {
	u, _ := domain.CreateUser("Original", nil)
	f := &fakeRepository{user: u, err: fmt.Errorf("stale version: %w", coreerrors.ErrConflict)}
	name := "Updated"
	_, err := NewUsersService(f).PatchUser(context.Background(), u.ID, domain.UserPatch{FullName: domain.Nullable[string]{Set: true, Value: &name}})
	if !errors.Is(err, coreerrors.ErrConflict) || f.updates != 1 {
		t.Fatalf("err=%v, updates=%d", err, f.updates)
	}
	if f.user.FullName != "Original" {
		t.Fatal("service mutated repository snapshot")
	}
}
