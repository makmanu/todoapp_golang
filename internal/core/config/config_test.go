package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestConfigEscapesCredentialsAndRejectsBadValues(t *testing.T) {
	t.Setenv("POSTGRES_USER", "test-user")
	t.Setenv("POSTGRES_PASSWORD", "p@ss:#/?")
	t.Setenv("POSTGRES_DB", "testdb")
	t.Setenv("POSTGRES_TIMEOUT", "2s")
	t.Setenv("POSTGRES_MAX_CONNS", "4")
	cfg, err := NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(cfg.PostgresURL)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := u.User.Password()
	if password != "p@ss:#/?" || u.Query().Get("search_path") != "public" {
		t.Fatal("invalid DSN")
	}
	t.Setenv("POSTGRES_TIMEOUT", "0s")
	if _, err := NewConfig(); err == nil {
		t.Fatal("accepted zero timeout")
	}
	t.Setenv("POSTGRES_TIMEOUT", "2s")
	t.Setenv("POSTGRES_PASSWORD", "")
	if _, err := NewConfig(); err == nil || !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatal(err)
	}
}
