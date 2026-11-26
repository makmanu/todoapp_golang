package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
	pgxpool "github.com/makmanu/golang_todoapp/internal/core/postgres/pool/pgx"
	"github.com/makmanu/golang_todoapp/internal/core/transport/http/server"
	"github.com/makmanu/golang_todoapp/internal/features/users/repository/postgres"
	"github.com/makmanu/golang_todoapp/internal/features/users/service"
	usershttp "github.com/makmanu/golang_todoapp/internal/features/users/transport/http"
)

// TestUsers requires a disposable, empty database. It never uses the application's .env.
func TestUsers(t *testing.T) {
	dsn := os.Getenv("TODOAPP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TODOAPP_TEST_DATABASE_URL to an empty disposable PostgreSQL database")
	}
	ctx := context.Background()
	p, err := pgxpool.New(ctx, dsn, 3*time.Second, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	migrate := func(name string) {
		t.Helper()
		sql, err := os.ReadFile(filepath.Join("..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	migrate("000001_init.up.sql")
	migrate("000002_user_phone.up.sql")
	repo := postgres.NewUsersRepository(p)
	app := newApp(service.NewUsersService(repo))
	ts := httptest.NewServer(app)
	defer ts.Close()
	client := http.Client{Timeout: 5 * time.Second}
	request := func(method, path, body string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+"/api/v1"+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, res.StatusCode, want, data)
		}
		if res.Header.Get("X-Request-ID") == "" {
			t.Fatal("missing request ID")
		}
		return data
	}
	decode := func(data []byte) usershttp.UserResponse {
		t.Helper()
		var u usershttp.UserResponse
		if err := json.Unmarshal(data, &u); err != nil {
			t.Fatal(err)
		}
		return u
	}
	if string(bytes.TrimSpace(request("GET", "/users", "", 200))) != "[]" {
		t.Fatal("empty list is not []")
	}
	request("POST", "/users", `{"full_name":"ab"}`, 400)
	request("POST", "/users", `{"full_name":"Name","phone_number":"+bad"}`, 400)
	u := decode(request("POST", "/users", `{"full_name":"Иван Иванов","phone_number":"+79991234567"}`, 201))
	if u.Version != 1 || u.PhoneNumber == nil {
		t.Fatalf("%+v", u)
	}
	path := "/users/" + u.ID
	got := decode(request("GET", path, "", 200))
	if got.FullName != u.FullName {
		t.Fatal(got)
	}
	got = decode(request("PATCH", path, `{"full_name":"Новое имя"}`, 200))
	if got.Version != 2 || got.PhoneNumber == nil || *got.PhoneNumber != "+79991234567" {
		t.Fatal("omitted phone changed")
	}
	got = decode(request("PATCH", path, `{"phone_number":null}`, 200))
	if got.Version != 3 || got.PhoneNumber != nil {
		t.Fatal("null phone not persisted")
	}
	got = decode(request("PATCH", path, `{"phone_number":"+79990000000"}`, 200))
	if got.Version != 4 || got.PhoneNumber == nil || *got.PhoneNumber != "+79990000000" {
		t.Fatal("phone update not persisted")
	}
	request("PATCH", path, `{"full_name":null}`, 400)
	request("PATCH", path, `{"full_name":"ab","phone_number":null}`, 400)
	got = decode(request("GET", path, "", 200))
	if got.Version != 4 || got.PhoneNumber == nil {
		t.Fatal("invalid patch changed persisted data")
	}
	request("GET", "/users?limit=-1", "", 400)
	request("GET", "/users?offset=-1", "", 400)
	if string(bytes.TrimSpace(request("GET", "/users?limit=0", "", 200))) != "[]" {
		t.Fatal("limit=0 ignored")
	}
	if string(bytes.TrimSpace(request("GET", "/users?offset=1", "", 200))) != "[]" {
		t.Fatal("offset ignored")
	}
	var list []usershttp.UserResponse
	if err := json.Unmarshal(request("GET", "/users?limit=1&offset=0", "", 200), &list); err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}

	// Both HTTP PATCH requests read the same version before either UPDATE runs.
	gated := &gatedRepository{UsersRepository: repo, ready: make(chan struct{})}
	raceServer := httptest.NewServer(newApp(service.NewUsersService(gated)))
	defer raceServer.Close()
	statuses := make(chan int, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			req, err := http.NewRequest("PATCH", raceServer.URL+"/api/v1"+path, bytes.NewBufferString(fmt.Sprintf(`{"full_name":"Writer %d"}`, i)))
			if err != nil {
				failures <- err
				return
			}
			res, err := client.Do(req)
			if err != nil {
				failures <- err
				return
			}
			defer res.Body.Close()
			_, _ = io.Copy(io.Discard, res.Body)
			statuses <- res.StatusCode
		}(i)
	}
	counts := map[int]int{}
	for i := 0; i < 2; i++ {
		select {
		case status := <-statuses:
			counts[status]++
		case err := <-failures:
			t.Fatal(err)
		case <-time.After(6 * time.Second):
			t.Fatal("concurrent PATCH timed out")
		}
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent PATCH statuses: %v", counts)
	}
	got = decode(request("GET", path, "", 200))
	if got.Version != 5 {
		t.Fatalf("concurrent patch incremented version incorrectly: %d", got.Version)
	}

	// A user referenced by a task cannot be deleted; the adapter maps FK violations.
	taskID := "00000000-0000-4000-8000-000000000001"
	if _, err := p.Exec(ctx, `INSERT INTO todoapp.tasks(id,title,author_user_id) VALUES($1,$2,$3)`, taskID, "Task", u.ID); err != nil {
		t.Fatal(err)
	}
	request("DELETE", path, "", 409)
	if _, err := p.Exec(ctx, "DELETE FROM todoapp.tasks WHERE id=$1", taskID); err != nil {
		t.Fatal(err)
	}
	if len(request("DELETE", path, "", 204)) != 0 {
		t.Fatal("DELETE returned a body")
	}
	request("DELETE", path, "", 404)
	request("GET", path, "", 404)
	request("PATCH", path, `{}`, 404)

	migrate("000002_user_phone.down.sql")
	var count int
	if err := p.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='todoapp' AND table_name='users' AND column_name='phone_number'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("phone rollback failed", err)
	}
	migrate("000002_user_phone.up.sql")
	request("POST", "/users", `{"full_name":"After rollback"}`, 201)
	migrate("000002_user_phone.down.sql")
	migrate("000001_init.down.sql")
}
func newApp(s usershttp.UsersService) http.Handler {
	r := server.NewAPIVersionRouter(server.ApiVersion1)
	r.AddRoutes(usershttp.NewUsersHTTPHandler(s).Routes()...)
	srv := server.NewHTTPServer("", time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.RegisterAPIRouters(r)
	return srv.Handler()
}

type gatedRepository struct {
	*postgres.UsersRepository
	reads atomic.Int32
	ready chan struct{}
}

func (r *gatedRepository) GetUser(ctx context.Context, id domain.UserID) (domain.User, error) {
	u, err := r.UsersRepository.GetUser(ctx, id)
	if err != nil {
		return u, err
	}
	if r.reads.Add(1) == 2 {
		close(r.ready)
	}
	select {
	case <-r.ready:
		return u, nil
	case <-ctx.Done():
		return domain.User{}, ctx.Err()
	}
}
