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
	// Assets, when configured, makes /assets/* redirect to presigned links
	// into a private bucket instead of serving internal/assets from disk.
	Assets       AssetsBucketConfig
	OpenAIAPIKey string
	OpenAIModel  string
	Grading      GradingConfig
	Speaking     SpeakingConfig
	// MediaDir holds recordings and examiner audio when no bucket is
	// configured (local development).
	MediaDir string
}

// SpeakingConfig tunes speaking grading and the examiner voice.
type SpeakingConfig struct {
	WhisperModel string
	// JudgeModel rates the band descriptors; empty means OPENAI_MODEL.
	// Rating is a judgement task, so a stronger model than the writing
	// default is worth its cost at four calls per test.
	JudgeModel    string
	TTSModel      string
	ExaminerVoice string
	// PronURL is the pronunciation service (services/pronunciation);
	// empty means pronunciation is always estimated.
	PronURL     string
	PronToken   string
	PronTimeout time.Duration
	// JobTimeout bounds one speaking grade: every answer is transcribed
	// and a sample waits on the pronunciation service, which runs on a
	// small free CPU VM.
	JobTimeout time.Duration
}

// AssetsBucketConfig points at a private S3-compatible bucket (Backblaze
// B2, R2) that mirrors internal/assets: audio/<id>.mp3, diagrams/<id>.jpg.
// Empty Bucket means serve from disk.
type AssetsBucketConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
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
	// TLS turns on verified TLS to the server (system CA bundle). Required
	// by managed MySQL like TiDB Cloud; off for the local compose MySQL.
	TLS bool
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
	cfg.App.Assets = AssetsBucketConfig{
		Endpoint:        strings.TrimRight(get("S3_ENDPOINT", ""), "/"),
		Region:          get("S3_REGION", ""),
		Bucket:          get("S3_BUCKET", ""),
		AccessKeyID:     get("S3_ACCESS_KEY_ID", ""),
		SecretAccessKey: get("S3_SECRET_ACCESS_KEY", ""),
	}
	if e := cfg.App.Assets.Endpoint; e != "" && !strings.Contains(e, "://") {
		cfg.App.Assets.Endpoint = "https://" + e
	}
	if a := cfg.App.Assets; a.Bucket != "" &&
		(a.Endpoint == "" || a.Region == "" || a.AccessKeyID == "" || a.SecretAccessKey == "") {
		return nil, fmt.Errorf("S3_BUCKET is set: S3_ENDPOINT, S3_REGION, S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are required too")
	}
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

	// Speaking
	cfg.App.Speaking.WhisperModel = get("WHISPER_MODEL", "whisper-1")
	cfg.App.Speaking.JudgeModel = get("SPEAKING_MODEL", cfg.App.OpenAIModel)
	cfg.App.Speaking.TTSModel = get("EXAMINER_TTS_MODEL", "gpt-4o-mini-tts")
	cfg.App.Speaking.ExaminerVoice = get("EXAMINER_VOICE", "sage")
	cfg.App.Speaking.PronURL = strings.TrimRight(get("PRON_SERVICE_URL", ""), "/")
	cfg.App.Speaking.PronToken = get("PRON_SERVICE_TOKEN", "")
	pronTimeout, err := getInt("PRON_SERVICE_TIMEOUT_SECONDS", 180)
	if err != nil {
		return nil, fmt.Errorf("PRON_SERVICE_TIMEOUT_SECONDS: %w", err)
	}
	cfg.App.Speaking.PronTimeout = time.Duration(pronTimeout) * time.Second
	speakingTimeout, err := getInt("SPEAKING_JOB_TIMEOUT_SECONDS", 720)
	if err != nil {
		return nil, fmt.Errorf("SPEAKING_JOB_TIMEOUT_SECONDS: %w", err)
	}
	cfg.App.Speaking.JobTimeout = time.Duration(speakingTimeout) * time.Second
	cfg.App.MediaDir = get("MEDIA_DIR", "data/media")

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
	dbTLS, err := getBool("DB_TLS", false)
	if err != nil {
		return nil, fmt.Errorf("DB_TLS: %w", err)
	}
	cfg.DB.TLS = dbTLS

	return cfg, nil
}

// func (c *Config) isDevelopmentt() bool { return c.App.Env == "development" }
