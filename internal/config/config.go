package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv   string `mapstructure:"APP_ENV"`
	Server   ServerConfig
	Database DatabaseConfig
	LogLevel string `mapstructure:"LOG_LEVEL"`
}

type ServerConfig struct {
	Port    string        `mapstructure:"SERVER_PORT"`
	Timeout time.Duration `mapstructure:"SERVER_TIMEOUT"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSL_MODE"`
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf(".env dosyası yüklenemedi: %w", err)
	}

	viper.AutomaticEnv()

	var cfg Config

	cfg.AppEnv = viper.GetString("APP_ENV")
	cfg.Server.Port = viper.GetString("SERVER_PORT")
	cfg.Server.Timeout = viper.GetDuration("SERVER_TIMEOUT")

	cfg.Database.Host = viper.GetString("DB_HOST")
	cfg.Database.Port = viper.GetString("DB_PORT")
	cfg.Database.User = viper.GetString("DB_USER")
	cfg.Database.Password = viper.GetString("DB_PASSWORD")
	cfg.Database.Name = viper.GetString("DB_NAME")
	cfg.Database.SSLMode = viper.GetString("DB_SSL_MODE")

	cfg.LogLevel = viper.GetString("LOG_LEVEL")

	return &cfg, nil
}
