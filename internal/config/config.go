package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv   string `mapstructure:"APP_ENV"`
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	LogLevel string `mapstructure:"LOG_LEVEL"`
}

type ServerConfig struct {
	Host         string `mapstructure:"SERVER_HOST"`
	Port         string `mapstructure:"SERVER_PORT"`
	ReadTimeout  int    `mapstructure:"SERVER_READ_TIMEOUT"`
	WriteTimeout int    `mapstructure:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout  int    `mapstructure:"SERVER_IDLE_TIMEOUT"`

	LoadBalancer LoadBalancerConfig `mapstructure:"load_balancer"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSL_MODE"`

	ReadReplicas []ReplicaConfig `mapstructure:"read_replicas"`

	MaxOpenConns    int `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime int `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

type ReplicaConfig struct {
	Host     string `mapstructure:"DB_READ_HOST_1"`
	Port     string `mapstructure:"DB_READ_PORT_1"`
	User     string `mapstructure:"DB_READ_USER_1"`
	Password string `mapstructure:"DB_READ_PASSWORD_1"`
	Name     string `mapstructure:"DB_READ_NAME_1"`
	SSLMode  string `mapstructure:"DB_READ_SSL_MODE_1"`
	Weight   int    `mapstructure:"DB_READ_WEIGHT_1"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`

	Cluster bool     `mapstructure:"REDIS_CLUSTER"`
	Nodes   []string `mapstructure:"REDIS_CLUSTER_NODES"`

	PoolSize     int `mapstructure:"REDIS_POOL_SIZE"`
	MinIdleConns int `mapstructure:"REDIS_MIN_IDLE_CONNS"`
}

type LoadBalancerConfig struct {
	Enabled             bool   `mapstructure:"LB_ENABLED"`
	Algorithm           string `mapstructure:"LB_ALGORITHM"`
	HealthCheckPath     string `mapstructure:"LB_HEALTH_CHECK_PATH"`
	HealthCheckInterval int    `mapstructure:"LB_HEALTH_CHECK_INTERVAL"`
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("UYARI: .env dosyası bulunamadı, çevresel değişkenler kullanılacak")
	}

	viper.AutomaticEnv()

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("SERVER_PORT", "8081")
	viper.SetDefault("SERVER_TIMEOUT", "30s")
	viper.SetDefault("LOG_LEVEL", "info")

	var cfg Config

	cfg.AppEnv = viper.GetString("APP_ENV")
	cfg.Server.Port = viper.GetString("SERVER_PORT")

	cfg.Database.Host = viper.GetString("DB_HOST")
	cfg.Database.Port = viper.GetString("DB_PORT")
	cfg.Database.User = viper.GetString("DB_USER")
	cfg.Database.Password = viper.GetString("DB_PASSWORD")
	cfg.Database.Name = viper.GetString("DB_NAME")
	cfg.Database.SSLMode = viper.GetString("DB_SSL_MODE")

	cfg.Redis.Host = viper.GetString("REDIS_HOST")
	cfg.Redis.Port = viper.GetString("REDIS_PORT")
	cfg.Redis.Password = viper.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = viper.GetInt("REDIS_DB")

	cfg.Redis.Cluster = viper.GetBool("REDIS_CLUSTER")
	cfg.Redis.Nodes = viper.GetStringSlice("REDIS_CLUSTER_NODES")
	cfg.Redis.PoolSize = viper.GetInt("REDIS_POOL_SIZE")
	cfg.Redis.MinIdleConns = viper.GetInt("REDIS_MIN_IDLE_CONNS")

	cfg.Server.Host = viper.GetString("SERVER_HOST")
	cfg.Server.ReadTimeout = viper.GetInt("SERVER_READ_TIMEOUT")
	cfg.Server.WriteTimeout = viper.GetInt("SERVER_WRITE_TIMEOUT")
	cfg.Server.IdleTimeout = viper.GetInt("SERVER_IDLE_TIMEOUT")

	cfg.Server.LoadBalancer.Enabled = viper.GetBool("LB_ENABLED")
	cfg.Server.LoadBalancer.Algorithm = viper.GetString("LB_ALGORITHM")
	cfg.Server.LoadBalancer.HealthCheckPath = viper.GetString("LB_HEALTH_CHECK_PATH")
	cfg.Server.LoadBalancer.HealthCheckInterval = viper.GetInt("LB_HEALTH_CHECK_INTERVAL")

	cfg.LogLevel = viper.GetString("LOG_LEVEL")

	return &cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsSlice(key, separator string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{}
	}
	return []string{}
}
