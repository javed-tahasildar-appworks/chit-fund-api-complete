package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	RateLimit   RateLimitConfig
	Razorpay    RazorpayConfig
}

type RazorpayConfig struct {
	KeyID         string // rzp_test_xxx or rzp_live_xxx
	KeySecret     string // Razorpay key secret
	WebhookSecret string // Razorpay webhook signing secret
	AccountNumber string // RazorpayX virtual account number (for payouts)
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int // minutes
	RefreshTokenTTL int // hours
}

type RateLimitConfig struct {
	RequestsPerMinute int
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("environment", "development")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.maxconns", 25)
	viper.SetDefault("database.minconns", 5)
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.accesstokenttl", 15)
	viper.SetDefault("jwt.refreshtokenttl", 24)
	viper.SetDefault("ratelimit.requestsperminute", 60)
	viper.SetDefault("razorpay.key_id", "")
	viper.SetDefault("razorpay.key_secret", "")
	viper.SetDefault("razorpay.webhook_secret", "")
	viper.SetDefault("razorpay.account_number", "")

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("No config file found, using defaults and environment variables")
	}

	return &Config{
		Environment: viper.GetString("environment"),
		Server: ServerConfig{
			Port: viper.GetInt("server.port"),
			Host: viper.GetString("server.host"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetInt("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			DBName:   viper.GetString("database.dbname"),
			SSLMode:  viper.GetString("database.sslmode"),
			MaxConns: int32(viper.GetInt("database.maxconns")),
			MinConns: int32(viper.GetInt("database.minconns")),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:          viper.GetString("jwt.secret"),
			AccessTokenTTL:  viper.GetInt("jwt.accesstokenttl"),
			RefreshTokenTTL: viper.GetInt("jwt.refreshtokenttl"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: viper.GetInt("ratelimit.requestsperminute"),
		},
		Razorpay: RazorpayConfig{
			KeyID:         viper.GetString("razorpay.key_id"),
			KeySecret:     viper.GetString("razorpay.key_secret"),
			WebhookSecret: viper.GetString("razorpay.webhook_secret"),
			AccountNumber: viper.GetString("razorpay.account_number"),
		},
	}
}
