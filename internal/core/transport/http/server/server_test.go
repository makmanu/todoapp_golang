package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownDrainsActiveRequest(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := NewHTTPServer("127.0.0.1:0", time.Second, log)
	entered, release := make(chan struct{}), make(chan struct{})
	s.RegisterRoutes(Route{Method: "GET", Path: "/slow", Handler: func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte("finished"))
	}})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln) }()
	response := make(chan error, 1)
	go func() {
		client := http.Client{Timeout: 3 * time.Second}
		r, err := client.Get("http://" + ln.Addr().String() + "/slow")
		if err == nil {
			defer r.Body.Close()
			var body []byte
			body, err = io.ReadAll(r.Body)
			if string(body) != "finished" {
				err = io.ErrUnexpectedEOF
			}
		}
		response <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		t.Fatalf("server exited before active request finished: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-response; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not complete")
	}
}
func TestShutdownDeadlineClosesConnection(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := NewHTTPServer("127.0.0.1:0", 20*time.Millisecond, log)
	entered := make(chan struct{})
	s.RegisterRoutes(Route{Method: "GET", Path: "/blocked", Handler: func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln) }()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		client := http.Client{Timeout: 2 * time.Second}
		r, _ := client.Get("http://" + ln.Addr().String() + "/blocked")
		if r != nil {
			r.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected shutdown timeout")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server hung")
	}
	<-clientDone
}
