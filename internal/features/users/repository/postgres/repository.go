package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
	"github.com/makmanu/golang_todoapp/internal/core/postgres/pool"
)

type UsersRepository struct{ pool pool.Pool }

func NewUsersRepository(p pool.Pool) *UsersRepository { return &UsersRepository{pool: p} }

func (r *UsersRepository) SaveUser(ctx context.Context, user domain.User) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()
	row := r.pool.QueryRow(ctx, `INSERT INTO todoapp.users (id, version, full_name, phone_number)
 VALUES ($1, $2, $3, $4) RETURNING id, version, full_name, phone_number`,
		string(user.ID), user.Version, user.FullName, user.PhoneNumber)
	var m UserModel
	if err := m.Scan(row); err != nil {
		return domain.User{}, mapError("save user", err)
	}
	return modelToDomain(m), nil
}
func (r *UsersRepository) GetUser(ctx context.Context, id domain.UserID) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()
	var m UserModel
	err := m.Scan(r.pool.QueryRow(ctx, `SELECT id, version, full_name, phone_number FROM todoapp.users WHERE id=$1`, string(id)))
	if errors.Is(err, pool.ErrNoRows) {
		return domain.User{}, fmt.Errorf("user %s: %w", id, coreerrors.ErrNotFound)
	}
	if err != nil {
		return domain.User{}, mapError("get user", err)
	}
	return modelToDomain(m), nil
}
func (r *UsersRepository) GetUsers(ctx context.Context, limit, offset *int) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT id, version, full_name, phone_number FROM todoapp.users ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError("query users", err)
	}
	defer rows.Close()
	users := make([]domain.User, 0)
	for rows.Next() {
		var m UserModel
		if err := m.Scan(rows); err != nil {
			return nil, mapError("scan user", err)
		}
		users = append(users, modelToDomain(m))
	}
	if err := rows.Err(); err != nil {
		return nil, mapError("read users", err)
	}
	return users, nil
}
func (r *UsersRepository) UpdateUser(ctx context.Context, user domain.User) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()
	row := r.pool.QueryRow(ctx, `UPDATE todoapp.users SET full_name=$1, phone_number=$2, version=version+1
 WHERE id=$3 AND version=$4 RETURNING id, version, full_name, phone_number`,
		user.FullName, user.PhoneNumber, string(user.ID), user.Version)
	var m UserModel
	err := m.Scan(row)
	if errors.Is(err, pool.ErrNoRows) {
		return domain.User{}, fmt.Errorf("user %s changed concurrently: %w", user.ID, coreerrors.ErrConflict)
	}
	if err != nil {
		return domain.User{}, mapError("update user", err)
	}
	return modelToDomain(m), nil
}
func (r *UsersRepository) DeleteUser(ctx context.Context, id domain.UserID) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM todoapp.users WHERE id=$1`, string(id))
	if err != nil {
		return mapError("delete user", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", id, coreerrors.ErrNotFound)
	}
	return nil
}
func mapError(operation string, err error) error {
	switch {
	case errors.Is(err, pool.ErrViolatesForeignKey), errors.Is(err, pool.ErrUniqueViolation):
		return fmt.Errorf("%s: %w: %w", operation, coreerrors.ErrConflict, err)
	case errors.Is(err, pool.ErrCheckViolation):
		return fmt.Errorf("%s: %w: %w", operation, coreerrors.ErrInvalidArgument, err)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
