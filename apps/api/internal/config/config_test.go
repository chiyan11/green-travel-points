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

func TestLoadDisablesDevHeaderByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AUTH_ALLOW_DEV_HEADER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthAllowDevHeader {
		t.Fatal("未显式开启时 AuthAllowDevHeader 必须为 false")
	}
}

func TestLoadAcceptsDevHeaderWhenExplicitlyEnabled(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AUTH_ALLOW_DEV_HEADER", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.AuthAllowDevHeader {
		t.Fatal("AUTH_ALLOW_DEV_HEADER=true 时 AuthAllowDevHeader 应为 true")
	}
}

func TestLoadRejectsDevHeaderInProd(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("AUTH_SECRET", "test-secret")
	t.Setenv("AUTH_ALLOW_DEV_HEADER", "true")

	if _, err := Load(); err == nil {
		t.Fatal("APP_ENV=prod 且开启开发身份头时应拒绝启动")
	}
}
