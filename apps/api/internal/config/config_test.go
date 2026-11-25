package config

import "testing"

func TestLoadRequiresAuthSecretInProd(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("AUTH_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("APP_ENV=prod 且 AUTH_SECRET 为空时应返回错误")
	}
}

func TestLoadAcceptsAuthSecretInProd(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("AUTH_SECRET", "test-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthSecret != "test-secret" {
		t.Fatalf("AuthSecret = %q", cfg.AuthSecret)
	}
}
