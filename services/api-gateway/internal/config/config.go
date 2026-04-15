package config

import (
	"os"
	"time"
)

type Config struct {
	Port string
	RedisAddr string
	ReadTimeout time.Duration
	WriteTimeout time.Duration
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return &Config{
		Port: port,
		RedisAddr: redisAddr,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}