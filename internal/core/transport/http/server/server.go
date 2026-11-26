package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/makmanu/golang_todoapp/internal/core/transport/http/middleware"
)

type APIVersion string

const ApiVersion1 APIVersion = "v1"

type Route struct {
	Method     string
	Path       string
	Handler    http.HandlerFunc
	Middleware []middleware.Middleware
}
type APIVersionRouter struct {
	version    APIVersion
	routes     []Route
	middleware []middleware.Middleware
}

func NewAPIVersionRouter(version APIVersion, m ...middleware.Middleware) *APIVersionRouter {
	return &APIVersionRouter{version: version, middleware: m}
}
func (r *APIVersionRouter) AddRoutes(routes ...Route) { r.routes = append(r.routes, routes...) }

type HTTPServer struct {
	server          *http.Server
	mux             *http.ServeMux
	shutdownTimeout time.Duration
	log             *slog.Logger
}

func NewHTTPServer(address string, shutdownTimeout time.Duration, log *slog.Logger) *HTTPServer {
	mux := http.NewServeMux()
	h := middleware.ChainMiddleware(mux, middleware.CORS, middleware.RequestID, middleware.Logger(log), middleware.Trace, middleware.Panic)
	return &HTTPServer{mux: mux, shutdownTimeout: shutdownTimeout, log: log, server: &http.Server{
		Addr: address, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20,
	}}
}
func (s *HTTPServer) RegisterRoutes(routes ...Route) {
	for _, r := range routes {
		s.mux.Handle(r.Method+" "+r.Path, middleware.ChainMiddleware(r.Handler, r.Middleware...))
	}
}
func (s *HTTPServer) RegisterAPIRouters(routers ...*APIVersionRouter) {
	for _, router := range routers {
		for _, route := range router.routes {
			route.Path = "/api/" + string(router.version) + route.Path
			route.Middleware = append(append([]middleware.Middleware{}, router.middleware...), route.Middleware...)
			s.RegisterRoutes(route)
		}
	}
}
func (s *HTTPServer) Handler() http.Handler { return s.server.Handler }
func (s *HTTPServer) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return s.Serve(ctx, listener)
}

// Serve owns listener and drains active requests when ctx is cancelled.
func (s *HTTPServer) Serve(ctx context.Context, listener net.Listener) error {
	done := make(chan error, 1)
	go func() { done <- s.server.Serve(listener) }()
	s.log.Info("HTTP server started", "address", listener.Addr().String())
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			_ = s.server.Close()
			<-done
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
		err := <-done
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}
