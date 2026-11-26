package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/makmanu/golang_todoapp/internal/core/logger"
	corehttp "github.com/makmanu/golang_todoapp/internal/core/transport/http"
)

type Middleware func(http.Handler) http.Handler

func ChainMiddleware(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Location")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type requestIDKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			var b [16]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}
func validRequestID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestLog := log.With("request_id", RequestIDFromContext(r.Context()), "url", r.URL.Path)
			next.ServeHTTP(w, r.WithContext(logger.WithContext(r.Context(), requestLog)))
		})
	}
}

type ResponseWriter struct {
	http.ResponseWriter
	Status int
	Bytes  int
}

func (w *ResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *ResponseWriter) WriteHeader(status int) {
	if w.Status != 0 {
		return
	}
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.Bytes += n
	return n, err
}
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &ResponseWriter{ResponseWriter: w}
		log := logger.FromContext(r.Context())
		log.Info("request started", "method", r.Method)
		defer func() {
			status := rw.Status
			if status == 0 {
				status = http.StatusOK
			}
			log.Info("request completed", "status", status, "bytes", rw.Bytes, "duration", time.Since(start))
		}()
		next.ServeHTTP(rw, r)
	})
}
func Panic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				logger.FromContext(r.Context()).Error("handler panic", "panic", recovered, "stack", string(debug.Stack()))
				if rw, ok := w.(*ResponseWriter); ok && rw.Status != 0 {
					panic(http.ErrAbortHandler)
				}
				corehttp.WriteJSON(w, http.StatusInternalServerError, corehttp.ErrorResponse{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
