package config

import (
	"reflect"
	"testing"
	"time"
)

const validSecret = "a-valid-secret-key-of-at-least-32-chars"

func baseEnv() map[string]string {
	return map[string]string{"SECRET_KEY": validSecret}
}

func TestLoadFromMapDefaults(t *testing.T) {
	cfg, err := loadFromMap(baseEnv())
	if err != nil {
		t.Fatalf("loadFromMap: %v", err)
	}

	if cfg.App.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.App.Env)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.App.Port)
	}
	if !cfg.App.AutoMigrate {
		t.Error("AutoMigrate should default to true")
	}
	if cfg.App.OpenAIModel != "gpt-4o-mini" {
		t.Errorf("OpenAIModel = %q, want gpt-4o-mini", cfg.App.OpenAIModel)
	}
	if cfg.DB.Host != "localhost" || cfg.DB.Port != 3306 {
		t.Errorf("DB host/port = %s:%d, want localhost:3306", cfg.DB.Host, cfg.DB.Port)
	}

	// Empty means same-origin, which is the default deployment. It must
	// not silently become "reject every origin" the way it used to.
	if len(cfg.App.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v, want empty by default", cfg.App.AllowedOrigins)
	}

	if cfg.App.Grading.Workers != 2 {
		t.Errorf("Grading.Workers = %d, want 2", cfg.App.Grading.Workers)
	}
	if cfg.App.Grading.PollInterval != 3*time.Second {
		t.Errorf("Grading.PollInterval = %v, want 3s", cfg.App.Grading.PollInterval)
	}
	if cfg.App.Grading.JobTimeout != 90*time.Second {
		t.Errorf("Grading.JobTimeout = %v, want 90s", cfg.App.Grading.JobTimeout)
	}
}

// An empty or weak SECRET_KEY means JWTs are signed with a guessable (or
// empty) key, so anyone can mint a token claiming role=admin. Booting
// anyway is not an option.
func TestLoadFromMapRequiresAStrongSecretKey(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{"missing entirely", ""},
		{"empty string", ""},
		{"far too short", "short"},
		{"one char below the minimum", "1234567890123456789012345678901"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{}
			if tc.secret != "" {
				env["SECRET_KEY"] = tc.secret
			}
			if _, err := loadFromMap(env); err == nil {
				t.Fatalf("expected SECRET_KEY %q to be rejected", tc.secret)
			}
		})
	}

	exactlyMinimum := "12345678901234567890123456789012" // 32 chars
	if _, err := loadFromMap(map[string]string{"SECRET_KEY": exactlyMinimum}); err != nil {
		t.Errorf("a %d-character secret should be accepted, got %v", len(exactlyMinimum), err)
	}
}

func TestLoadFromMapParsesAllowedOrigins(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"single origin", "https://a.example", []string{"https://a.example"}},
		{"comma separated", "https://a.example,https://b.example", []string{"https://a.example", "https://b.example"}},
		{"whitespace is trimmed", " https://a.example , https://b.example ", []string{"https://a.example", "https://b.example"}},
		{"empty entries are dropped", "https://a.example,,  ,", []string{"https://a.example"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := baseEnv()
			env["ALLOWED_ORIGINS"] = tc.raw

			cfg, err := loadFromMap(env)
			if err != nil {
				t.Fatalf("loadFromMap: %v", err)
			}
			if !reflect.DeepEqual(cfg.App.AllowedOrigins, tc.want) {
				t.Errorf("AllowedOrigins = %#v, want %#v", cfg.App.AllowedOrigins, tc.want)
			}
		})
	}
}

func TestLoadFromMapRejectsMalformedNumbers(t *testing.T) {
	for _, key := range []string{"PORT", "DB_PORT", "GRADING_WORKERS", "GRADING_POLL_SECONDS", "GRADING_JOB_TIMEOUT_SECONDS"} {
		t.Run(key, func(t *testing.T) {
			env := baseEnv()
			env[key] = "not-a-number"
			if _, err := loadFromMap(env); err == nil {
				t.Errorf("expected a malformed %s to be rejected", key)
			}
		})
	}
}

func TestLoadFromMapRejectsMalformedBool(t *testing.T) {
	env := baseEnv()
	env["AUTO_MIGRATE"] = "yes-please"
	if _, err := loadFromMap(env); err == nil {
		t.Error("expected a malformed AUTO_MIGRATE to be rejected")
	}
}

func TestLoadFromMapReadsOverrides(t *testing.T) {
	env := baseEnv()
	env["APP_ENV"] = "production"
	env["PORT"] = "9000"
	env["AUTO_MIGRATE"] = "false"
	env["DB_HOST"] = "mysql"
	env["DB_NAME"] = "ielts_prod"
	env["GRADING_WORKERS"] = "8"

	cfg, err := loadFromMap(env)
	if err != nil {
		t.Fatalf("loadFromMap: %v", err)
	}
	if cfg.App.Env != "production" || cfg.App.Port != 9000 || cfg.App.AutoMigrate {
		t.Errorf("app overrides not applied: %+v", cfg.App)
	}
	if cfg.DB.Host != "mysql" || cfg.DB.Name != "ielts_prod" {
		t.Errorf("db overrides not applied: %+v", cfg.DB)
	}
	if cfg.App.Grading.Workers != 8 {
		t.Errorf("Grading.Workers = %d, want 8", cfg.App.Grading.Workers)
	}
}
