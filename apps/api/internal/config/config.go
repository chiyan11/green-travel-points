package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 汇总服务运行所需的全部配置，来源为环境变量，附开发环境默认值。
// 生产环境务必覆盖 DB_DSN 与 APP_ENV。
type Config struct {
	Env     string // dev / test / prod
	AppName string

	HTTPAddr            string
	HTTPRequestTimeout  time.Duration
	HTTPShutdownTimeout time.Duration

	DatabaseDSN string
	AuthSecret  string
	// AuthAllowDevHeader 控制是否接受 X-User-Id 开发身份头；默认关闭，APP_ENV=prod 开即拒启。
	AuthAllowDevHeader bool
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
}

// DefaultDatabaseDSN 与 deploy/docker-compose.yml 中的 MySQL 服务保持一致。
const DefaultDatabaseDSN = "greentravel:greentravel@tcp(127.0.0.1:3306)/green_travel_points?parseTime=true&loc=Local&charset=utf8mb4"

// Load 读取环境变量并做基本校验。
func Load() (Config, error) {
	env := getEnv("APP_ENV", "dev")
	authSecret := getEnv("AUTH_SECRET", "")
	if env != "prod" && authSecret == "" {
		authSecret = "dev-only-change-me"
	}
	cfg := Config{
		Env:                 env,
		AppName:             getEnv("APP_NAME", "green-travel-points-api"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		HTTPRequestTimeout:  getEnvDuration("HTTP_REQUEST_TIMEOUT", 15*time.Second),
		HTTPShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseDSN:         getEnv("DB_DSN", DefaultDatabaseDSN),
		AuthSecret:          authSecret,
		AuthAllowDevHeader:  getEnvBool("AUTH_ALLOW_DEV_HEADER", false),
		DBMaxOpenConns:      getEnvInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:      getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:   getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("config: HTTP_ADDR 不能为空")
	}
	if cfg.DatabaseDSN == "" {
		return Config{}, fmt.Errorf("config: DB_DSN 不能为空")
	}
	if cfg.Env == "prod" && cfg.AuthSecret == "" {
		return Config{}, fmt.Errorf("config: AUTH_SECRET 不能为空（APP_ENV=prod）")
	}
	if cfg.Env == "prod" && cfg.AuthAllowDevHeader {
		return Config{}, fmt.Errorf("config: APP_ENV=prod 时不允许开启 AUTH_ALLOW_DEV_HEADER")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
