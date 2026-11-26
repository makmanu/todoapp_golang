package http

import (
	"fmt"
	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coretypes "github.com/makmanu/golang_todoapp/internal/core/transport/http/types"
)

type CreateUserRequest struct {
	FullName    string  `json:"full_name" validate:"required"`
	PhoneNumber *string `json:"phone_number"`
}
type PatchUserRequest struct {
	FullName    coretypes.Nullable[string] `json:"full_name"`
	PhoneNumber coretypes.Nullable[string] `json:"phone_number"`
}

func (r *PatchUserRequest) Validate() error {
	if r.FullName.Set && r.FullName.Value == nil {
		return fmt.Errorf("full_name cannot be null")
	}
	return nil
}
func (r PatchUserRequest) ToDomain() domain.UserPatch {
	return domain.UserPatch{FullName: r.FullName.ToDomain(), PhoneNumber: r.PhoneNumber.ToDomain()}
}

type UserResponse struct {
	ID          string  `json:"id"`
	Version     int64   `json:"version"`
	FullName    string  `json:"full_name"`
	PhoneNumber *string `json:"phone_number"`
}

func toResponse(u domain.User) UserResponse {
	return UserResponse{ID: string(u.ID), Version: u.Version, FullName: u.FullName, PhoneNumber: u.PhoneNumber}
}
