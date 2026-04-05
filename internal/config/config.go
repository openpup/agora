package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig            `mapstructure:"server"`
	Database DatabaseConfig          `mapstructure:"database"`
	Redis    RedisConfig             `mapstructure:"redis"`
	NATS     NATSConfig              `mapstructure:"nats"`
	Auth     AuthConfig              `mapstructure:"auth"`
	Markets  map[string]MarketConfig `mapstructure:"markets"`
	Workers  WorkersConfig           `mapstructure:"workers"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int32         `mapstructure:"max_open_conns"`
	MaxIdleConns    int32         `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type NATSConfig struct {
	URL            string `mapstructure:"url"`
	StreamReplicas int    `mapstructure:"stream_replicas"`
}

type AuthConfig struct {
	APIKeyPrefix     string `mapstructure:"api_key_prefix"`
	JWTSecret        string `mapstructure:"jwt_secret"`
	RateLimitPerMin  int    `mapstructure:"rate_limit_per_min"`
	IdempotencyTTL   string `mapstructure:"idempotency_ttl"`
	APIKeyHeaderName string `mapstructure:"api_key_header_name"`
}

type MarketConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	DataSource   string `mapstructure:"data_source"`
	SyncInterval string `mapstructure:"sync_interval"`
}

type WorkersConfig struct {
	TrustCalculator WorkerIntervalConfig `mapstructure:"trust_calculator"`
	SignalVerifier  WorkerIntervalConfig `mapstructure:"signal_verifier"`
	MarketDataSync  WorkerEnabledConfig  `mapstructure:"market_data_sync"`
}

type WorkerIntervalConfig struct {
	Interval time.Duration `mapstructure:"interval"`
}

type WorkerEnabledConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("auth.api_key_prefix", "ak_")
	v.SetDefault("auth.rate_limit_per_min", 1000)
	v.SetDefault("auth.idempotency_ttl", "24h")
	v.SetDefault("auth.api_key_header_name", "X-Agent-Key")
	v.SetDefault("workers.trust_calculator.interval", "5m")
	v.SetDefault("workers.signal_verifier.interval", "1m")
	v.SetDefault("workers.market_data_sync.interval", "1m")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config.Load read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config.Load unmarshal: %w", err)
	}

	return &cfg, nil
}
