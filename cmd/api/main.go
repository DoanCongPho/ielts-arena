package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/feature/profile"
	"github/DoanCongPho/game-arena/internal/feature/progression"
	"github/DoanCongPho/game-arena/internal/platform"
	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/config"
	"github/DoanCongPho/game-arena/internal/platform/database"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/middleware"
	"github/DoanCongPho/game-arena/internal/platform/storage"
	"github/DoanCongPho/game-arena/migrations"

	"github.com/gorilla/mux"
)

func helloWorld(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hello, world!\r\n")) }

func checkhealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func main() {
	// Needs no config or database, so it runs before either is loaded.
	if len(os.Args) > 1 && os.Args[1] == "validate-test" {
		os.Exit(ielts_test.RunValidateCmd(os.Args[2:]))
	}
	cfg := config.MustLoad()
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(database.RunMigrateCmd(cfg, migrations.FS, os.Args[2:]))
	}
	// AUTO_MIGRATE was parsed into config and then never read by anything,
	// so a fresh database booted with no tables and every request failed
	// against a schema that was never created. Run pending migrations
	// before opening the runtime pool.
	if cfg.App.AutoMigrate {
		if err := database.Migrate(&cfg.DB, migrations.FS); err != nil {
			log.Fatalf("auto-migrate: %v", err)
		}
		log.Println("migrations: up to date")
	}

	plat, err := platform.Build(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "import-test" {
		code := ielts_test.RunImportCmd(plat.DB, os.Args[2:])
		_ = plat.Close(context.Background())
		os.Exit(code)
	}
	defer func() { _ = plat.Close(context.Background()) }()

	auth.Init(cfg.App.SecretKey)

	r := mux.NewRouter()
	r.HandleFunc("/", helloWorld)
	r.HandleFunc("/health", checkhealth)

	// Static assets (e.g. Writing Task 1 chart images) — public, no auth.
	// With S3_BUCKET set (production), the files live in a private bucket and
	// this redirects to a short-lived presigned link, so stored URLs like
	// /assets/audio/x.mp3 keep working. presignAsset gives the writing
	// grader the same links for chart images; nil means read from disk.
	var presignAsset func(key string) (string, error)
	if a := cfg.App.Assets; a.Bucket != "" {
		bucket := &storage.Bucket{
			Endpoint: a.Endpoint, Region: a.Region, Name: a.Bucket,
			AccessKeyID: a.AccessKeyID, SecretAccessKey: a.SecretAccessKey,
		}
		presignAsset = func(key string) (string, error) {
			return bucket.PresignGet(key, 15*time.Minute, time.Now())
		}
		r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			link, err := bucket.PresignGet(req.URL.Path, time.Hour, time.Now())
			if err != nil {
				http.Error(w, "asset unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, req, link, http.StatusFound)
		})))
	} else {
		r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("internal/assets"))))
	}

	// Protected routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.RequireAuth)

	// -- Auth --
	authRepo := auth.NewUserRepository(plat.DB)
	authSvc := auth.NewService(authRepo)
	pubAPI := r.PathPrefix("/api").Subrouter()
	auth.NewHandler(authSvc).MountRoutes(pubAPI)

	// --progression--
	// Game progress (xp, level, rank). Shares the users
	// table with auth but owns a disjoint set of its columns.
	progRepo := progression.NewRepository(plat.DB)
	progSvc := progression.NewService(progRepo)

	// --ielts_test--
	llmClient := llm.NewClient(cfg.App.OpenAIAPIKey, cfg.App.OpenAIModel)
	grader := ielts_test.NewOpenAIGrader(llmClient, ielts_test.NewAssetImageResolver("internal/assets", presignAsset))

	testRepo := ielts_test.NewRepository(plat.DB)
	// progRepo satisfies ielts_test.XPGranter structurally — grading knows
	// only "something that can award XP", not the progression package.
	testSvc := ielts_test.NewService(testRepo, grader, progRepo)
	ielts_test.NewHandler(testSvc).MountRoutes(api)

	// Grading runs here, not in the request that submitted the answer:
	// writing/speaking need a slow paid API call, and a fixed-size pool
	// caps how many of those can be in flight at once no matter how many
	// submissions arrive. workerCtx is cancelled before the HTTP server
	// shuts down so in-flight grades return to the queue.
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	workers := ielts_test.StartGradingWorkers(workerCtx, testSvc, ielts_test.WorkerConfig{
		Count:        cfg.App.Grading.Workers,
		PollInterval: cfg.App.Grading.PollInterval,
		JobTimeout:   cfg.App.Grading.JobTimeout,
	})

	// --profile--
	profileSvc := profile.NewService(authRepo, progSvc)
	// mounted on /api subrouter, RequireAuth already applies
	profile.NewHandler(profileSvc).MountRoutes(api)

	// CORS is off unless ALLOWED_ORIGINS says otherwise. The default
	// deployment puts the SPA and this API behind one nginx, so every
	// request is same-origin and there is nothing to negotiate.
	var handler http.Handler = r
	if cors := middleware.NewCORS(cfg.App.AllowedOrigins); cors != nil {
		handler = cors.Handler(r)
		log.Printf("cors: enabled for %v", cfg.App.AllowedOrigins)
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failure: %v", err)
		}
	}()
	log.Printf("api listening on %s", addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}

	// Stop accepting new grading jobs and wait for in-flight ones. A grade
	// interrupted here leaves its submission claimed; the lease expires
	// and another worker (or the next boot) reclaims it, so nothing is
	// silently dropped.
	stopWorkers()
	workers.Wait()
	log.Println("shutdown complete")
}
