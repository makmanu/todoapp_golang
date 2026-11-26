package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
	"github.com/makmanu/golang_todoapp/internal/core/transport/http/server"
)

type stubService struct {
	err   error
	calls int
	patch domain.UserPatch
}

func (s *stubService) CreateUser(context.Context, string, *string) (domain.User, error) {
	s.calls++
	return domain.User{}, s.err
}
func (s *stubService) GetUser(context.Context, domain.UserID) (domain.User, error) {
	s.calls++
	return domain.User{}, s.err
}
func (s *stubService) GetUsers(context.Context, *int, *int) ([]domain.User, error) {
	s.calls++
	return nil, s.err
}
func (s *stubService) DeleteUser(context.Context, domain.UserID) error { s.calls++; return s.err }
func (s *stubService) PatchUser(_ context.Context, _ domain.UserID, p domain.UserPatch) (domain.User, error) {
	s.calls++
	s.patch = p
	return domain.User{}, s.err
}
func handler(s UsersService) http.Handler {
	h := NewUsersHTTPHandler(s)
	r := server.NewAPIVersionRouter(server.ApiVersion1)
	r.AddRoutes(h.Routes()...)
	srv := server.NewHTTPServer("", time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.RegisterAPIRouters(r)
	return srv.Handler()
}
func TestTransportValidation(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/users", ""},
		{"POST", "/users", "null"},
		{"POST", "/users", `{"full_name":123}`},
		{"POST", "/users", `{"unknown":true,"full_name":"Name"}`},
		{"POST", "/users", `{"full_name":"Name"}{}`},
		{"POST", "/users", `{}`},
		{"POST", "/users", `{"full_name":"` + strings.Repeat("x", 1<<20) + `"}`},
		{"PATCH", "/users/" + id, `{"full_name":null}`},
		{"PATCH", "/users/" + id, `{"phone_number":42}`},
		{"GET", "/users/not-a-uuid", ""},
		{"GET", "/users?limit=abc", ""},
		{"GET", "/users?offset=", ""},
		{"GET", "/users?limit=1&limit=2", ""},
		{"GET", "/users?limit=9999999999999999999999999999", ""},
		{"GET", "/users?limit=%zz", ""},
	} {
		t.Run(tc.method+" "+tc.path+" "+tc.body[:min(len(tc.body), 30)], func(t *testing.T) {
			s := &stubService{}
			w := httptest.NewRecorder()
			handler(s).ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1"+tc.path, strings.NewReader(tc.body)))
			if w.Code != 400 || s.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, s.calls, w.Body.String())
			}
		})
	}
}
func TestErrorMappingDoesNotLeakInternalDetails(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{coreerrors.ErrInvalidArgument, 400}, {coreerrors.ErrNotFound, 404}, {coreerrors.ErrConflict, 409}, {fmt.Errorf("database password=secret"), 500},
	} {
		s := &stubService{err: fmt.Errorf("wrapped: %w", tc.err)}
		w := httptest.NewRecorder()
		handler(s).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/users", nil))
		if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
func TestHTTPPatchPreservesAllThreeStates(t *testing.T) {
	for _, tc := range []struct {
		body          string
		set, hasValue bool
	}{
		{`{}`, false, false}, {`{"phone_number":null}`, true, false}, {`{"phone_number":"+79991234567"}`, true, true},
	} {
		s := &stubService{}
		w := httptest.NewRecorder()
		handler(s).ServeHTTP(w, httptest.NewRequest("PATCH", "/api/v1/users/00000000-0000-4000-8000-000000000001", strings.NewReader(tc.body)))
		if w.Code != 200 || s.patch.PhoneNumber.Set != tc.set || (s.patch.PhoneNumber.Value != nil) != tc.hasValue {
			t.Fatalf("%s: %+v", tc.body, s.patch)
		}
	}
}
func TestEmptyListIsArrayAndDeleteHasNoBody(t *testing.T) {
	s := &stubService{}
	w := httptest.NewRecorder()
	handler(s).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/users", nil))
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	handler(s).ServeHTTP(w, httptest.NewRequest("DELETE", "/api/v1/users/00000000-0000-4000-8000-000000000001", nil))
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
