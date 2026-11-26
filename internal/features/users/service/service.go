package service

import (
	"context"
	"fmt"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
)

type UsersRepository interface {
	SaveUser(context.Context, domain.User) (domain.User, error)
	GetUser(context.Context, domain.UserID) (domain.User, error)
	GetUsers(context.Context, *int, *int) ([]domain.User, error)
	UpdateUser(context.Context, domain.User) (domain.User, error)
	DeleteUser(context.Context, domain.UserID) error
}

type UsersService struct{ usersRepository UsersRepository }

func NewUsersService(repository UsersRepository) *UsersService {
	return &UsersService{usersRepository: repository}
}

func (s *UsersService) CreateUser(ctx context.Context, fullName string, phoneNumber *string) (domain.User, error) {
	user, err := domain.CreateUser(fullName, phoneNumber)
	if err != nil {
		return domain.User{}, err
	}
	if err := user.Validate(); err != nil {
		return domain.User{}, fmt.Errorf("validate user: %w", err)
	}
	user, err = s.usersRepository.SaveUser(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("save user: %w", err)
	}
	return user, nil
}

func (s *UsersService) GetUser(ctx context.Context, id domain.UserID) (domain.User, error) {
	if _, err := domain.ParseUserID(string(id)); err != nil {
		return domain.User{}, err
	}
	user, err := s.usersRepository.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *UsersService) GetUsers(ctx context.Context, limit, offset *int) ([]domain.User, error) {
	if limit != nil && *limit < 0 || offset != nil && *offset < 0 {
		return nil, fmt.Errorf("limit and offset must be nonnegative: %w", coreerrors.ErrInvalidArgument)
	}
	users, err := s.usersRepository.GetUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *UsersService) DeleteUser(ctx context.Context, id domain.UserID) error {
	if _, err := domain.ParseUserID(string(id)); err != nil {
		return err
	}
	if err := s.usersRepository.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *UsersService) PatchUser(ctx context.Context, id domain.UserID, patch domain.UserPatch) (domain.User, error) {
	if err := patch.Validate(); err != nil {
		return domain.User{}, fmt.Errorf("validate patch: %w", err)
	}
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if err := user.ApplyPatch(patch); err != nil {
		return domain.User{}, fmt.Errorf("apply patch: %w", err)
	}
	user, err = s.usersRepository.UpdateUser(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}
