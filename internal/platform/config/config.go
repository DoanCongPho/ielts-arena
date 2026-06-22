package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App AppConfig
	DB  DBConfig
}

type AppConfig struct {
	Env         string
	Port        int
	AutoMigrate bool
}
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func MustLoad() *Config {
	_ = godotenv.Load()
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	cfg, err := loadFromMap(env)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg

}

func loadFromMap(env map[string]string) (*Config, error) {
	get := func(k, def string) string {
		if v, ok := env[k]; ok && v != "" {
			return v
		}
		return def
	}
	getInt := func(k string, def int) (int, error) {
		v := get(k, "")
		if v == "" {
			return def, nil
		}
		return strconv.Atoi(v)
	}
	getBool := func(k string, def bool) (bool, error) {
		v := get(k, "")
		if v == "" {
			return def, nil
		}
		return strconv.ParseBool(v)
	}
	cfg := &Config{}
	cfg.App.Env = get("APP_ENV", "development")
	AutoMigrate, err := getBool("AUTO_MIGRATE", true)
	if err != nil {
		return nil, fmt.Errorf("AUTO MIGRATE: %w", err)
	}
	cfg.App.AutoMigrate = AutoMigrate
	port, err := getInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("PORT: %w", err)
	}
	cfg.App.Port = port

	// DB
	cfg.DB.Host = get("DB_HOST", "localhost")
	dbPort, err := getInt("DB_PORT", 3306)
	if err != nil {
		return nil, fmt.Errorf("DB_PORT: %w", err)
	}
	cfg.DB.Port = dbPort
	cfg.DB.User = get("DB_USER", "root")
	cfg.DB.Password = get("DB_PASSWORD", "")
	cfg.DB.Name = get("DB_NAME", " tutor-portal-test")

	return cfg, nil
}

// func (c *Config) isDevelopmentt() bool { return c.App.Env == "development" }
