package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
	corehttp "github.com/makmanu/golang_todoapp/internal/core/transport/http"
	"github.com/makmanu/golang_todoapp/internal/core/transport/http/server"
)

type UsersService interface {
	CreateUser(context.Context, string, *string) (domain.User, error)
	GetUser(context.Context, domain.UserID) (domain.User, error)
	GetUsers(context.Context, *int, *int) ([]domain.User, error)
	DeleteUser(context.Context, domain.UserID) error
	PatchUser(context.Context, domain.UserID, domain.UserPatch) (domain.User, error)
}
type UsersHTTPHandler struct{ usersService UsersService }

func NewUsersHTTPHandler(s UsersService) *UsersHTTPHandler { return &UsersHTTPHandler{usersService: s} }
func (h *UsersHTTPHandler) Routes() []server.Route {
	return []server.Route{
		{Method: http.MethodPost, Path: "/users", Handler: h.CreateUser},
		{Method: http.MethodGet, Path: "/users", Handler: h.GetUsers},
		{Method: http.MethodGet, Path: "/users/{id}", Handler: h.GetUser},
		{Method: http.MethodDelete, Path: "/users/{id}", Handler: h.DeleteUser},
		{Method: http.MethodPatch, Path: "/users/{id}", Handler: h.PatchUser},
	}
}
func (h *UsersHTTPHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var dto CreateUserRequest
	if err := corehttp.DecodeAndValidateRequest(r, &dto); err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	user, err := h.usersService.CreateUser(r.Context(), dto.FullName, dto.PhoneNumber)
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	w.Header().Set("Location", r.URL.Path+"/"+string(user.ID))
	corehttp.WriteJSON(w, http.StatusCreated, toResponse(user))
}
func (h *UsersHTTPHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUserID(r.PathValue("id"))
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	user, err := h.usersService.GetUser(r.Context(), id)
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	corehttp.WriteJSON(w, http.StatusOK, toResponse(user))
}
func (h *UsersHTTPHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUserID(r.PathValue("id"))
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	if err := h.usersService.DeleteUser(r.Context(), id); err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *UsersHTTPHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUserID(r.PathValue("id"))
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	var dto PatchUserRequest
	if err := corehttp.DecodeAndValidateRequest(r, &dto); err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	user, err := h.usersService.PatchUser(r.Context(), id, dto.ToDomain())
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	corehttp.WriteJSON(w, http.StatusOK, toResponse(user))
}
func (h *UsersHTTPHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		corehttp.WriteError(w, r, fmt.Errorf("invalid query: %w: %w", coreerrors.ErrInvalidArgument, err))
		return
	}
	limit, err := pagination(query, "limit")
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	offset, err := pagination(query, "offset")
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	users, err := h.usersService.GetUsers(r.Context(), limit, offset)
	if err != nil {
		corehttp.WriteError(w, r, err)
		return
	}
	result := make([]UserResponse, 0, len(users))
	for _, user := range users {
		result = append(result, toResponse(user))
	}
	corehttp.WriteJSON(w, http.StatusOK, result)
}
func pagination(values url.Values, key string) (*int, error) {
	items, exists := values[key]
	if !exists {
		return nil, nil
	}
	if len(items) != 1 || items[0] == "" {
		return nil, fmt.Errorf("%s must be one integer: %w", key, coreerrors.ErrInvalidArgument)
	}
	n, err := strconv.Atoi(items[0])
	if err != nil {
		return nil, fmt.Errorf("%s must be an integer: %w: %w", key, coreerrors.ErrInvalidArgument, err)
	}
	return &n, nil
}
