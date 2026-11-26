package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/makmanu/golang_todoapp/internal/core/logger"
)

func TestMiddlewareOrderAndPanicResponse(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	h := ChainMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) != "client-123" {
			t.Error("missing request id")
		}
		logger.FromContext(r.Context()).Info("handler entered")
		panic("private failure")
	}), CORS, RequestID, Logger(log), Trace, Panic)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("X-Request-ID", "client-123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 500 || strings.Contains(w.Body.String(), "private") || w.Header().Get("X-Request-ID") != "client-123" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"request_id":"client-123"`, `"url":"/api/v1/users"`, `"status":500`, "handler entered"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("missing %s in logs: %s", want, logs.String())
		}
	}
}
func TestCORSPreflightShortCircuits(t *testing.T) {
	h := ChainMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("preflight reached handler") }), CORS)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil))
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal(w)
	}
}
func TestResponseWriterKeepsFirstStatus(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: w}
	rw.WriteHeader(201)
	rw.WriteHeader(500)
	_, _ = rw.Write([]byte("ok"))
	if rw.Status != 201 || w.Code != 201 || rw.Bytes != 2 {
		t.Fatal(rw)
	}
	if rw.Unwrap() != w {
		t.Fatal("unwrap lost original writer")
	}
}
