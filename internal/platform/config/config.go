package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// minSecretKeyLen is 32 bytes = 256 bits, matching HS256's block size.
// A shorter key weakens the signature; an empty one removes it entirely.
const minSecretKeyLen = 32

type Config struct {
	App AppConfig
	DB  DBConfig
}

type AppConfig struct {
	Env         string
	Port        int
	AutoMigrate bool
	SecretKey   []byte
	// AllowedOrigins is empty for the default same-origin deployment (the
	// SPA and this API behind one nginx). Set it only when the browser
	// loads the frontend from a different origin than this API.
	AllowedOrigins []string
	OpenAIAPIKey   string
	OpenAIModel    string
	Grading        GradingConfig
}

// GradingConfig tunes the background grading worker pool. Workers is also
// the ceiling on concurrent OpenAI calls, so it doubles as the cost and
// rate control for LLM grading.
type GradingConfig struct {
	Workers      int
	PollInterval time.Duration
	JobTimeout   time.Duration
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
	// No default. An unset SECRET_KEY used to fall back to "", which means
	// every JWT was signed with an empty key — anyone could mint a token
	// with "role":"admin" and the server would accept it. Refusing to boot
	// is the only safe behaviour.
	secret := get("SECRET_KEY", "")
	if len(secret) < minSecretKeyLen {
		return nil, fmt.Errorf(
			"SECRET_KEY must be set and at least %d characters (got %d) — generate one with: openssl rand -base64 48",
			minSecretKeyLen, len(secret))
	}
	cfg.App.SecretKey = []byte(secret)
	if raw := get("ALLOWED_ORIGINS", ""); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				cfg.App.AllowedOrigins = append(cfg.App.AllowedOrigins, o)
			}
		}
	}
	cfg.App.OpenAIAPIKey = get("OPENAI_API_KEY", "")
	cfg.App.OpenAIModel = get("OPENAI_MODEL", "gpt-4o-mini")

	// Grading workers
	gradingWorkers, err := getInt("GRADING_WORKERS", 2)
	if err != nil {
		return nil, fmt.Errorf("GRADING_WORKERS: %w", err)
	}
	cfg.App.Grading.Workers = gradingWorkers
	pollSeconds, err := getInt("GRADING_POLL_SECONDS", 3)
	if err != nil {
		return nil, fmt.Errorf("GRADING_POLL_SECONDS: %w", err)
	}
	cfg.App.Grading.PollInterval = time.Duration(pollSeconds) * time.Second
	jobTimeoutSeconds, err := getInt("GRADING_JOB_TIMEOUT_SECONDS", 90)
	if err != nil {
		return nil, fmt.Errorf("GRADING_JOB_TIMEOUT_SECONDS: %w", err)
	}
	cfg.App.Grading.JobTimeout = time.Duration(jobTimeoutSeconds) * time.Second

	// DB
	cfg.DB.Host = get("DB_HOST", "localhost")
	dbPort, err := getInt("DB_PORT", 3306)
	if err != nil {
		return nil, fmt.Errorf("DB_PORT: %w", err)
	}
	cfg.DB.Port = dbPort
	cfg.DB.User = get("DB_USER", "root")
	cfg.DB.Password = get("DB_PASSWORD", "")
	cfg.DB.Name = get("DB_NAME", "ielts_arena")

	return cfg, nil
}

// func (c *Config) isDevelopmentt() bool { return c.App.Env == "development" }
