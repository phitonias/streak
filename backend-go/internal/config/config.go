package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	MongoDB  MongoDBConfig
	JWT      JWTConfig
	RTMP     RTMPConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
}

type MongoDBConfig struct {
	URL      string
	Database string
}

type JWTConfig struct {
	Secret        string
	RefreshSecret string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

type RTMPConfig struct {
	ServerURL string
}

type CORSConfig struct {
	AllowOrigins []string
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "3001"),
			Env:  getEnv("NODE_ENV", "development"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgresql://streak_user:streak_password@localhost:5432/streak"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		MongoDB: MongoDBConfig{
			URL:      getEnv("MONGODB_URL", "mongodb://streak_admin:streak_password@localhost:27017/streak_chat?authSource=admin"),
			Database: "streak_chat",
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "your-super-secret-refresh-key"),
			Expiry:        parseDuration(getEnv("JWT_EXPIRY", "15m"), 15*time.Minute),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"), 168*time.Hour),
		},
		RTMP: RTMPConfig{
			ServerURL: getEnv("RTMP_SERVER_URL", "rtmp://localhost:1935/live"),
		},
		CORS: CORSConfig{
			AllowOrigins: []string{getEnv("CORS_ORIGIN", "http://localhost:3000")},
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func parseDuration(s string, defaultDuration time.Duration) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Warning: Invalid duration format '%s', using default", s)
		return defaultDuration
	}
	return duration
}
