package main

import (
	"context"
	"errors"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform"
	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/config"
	"github/DoanCongPho/game-arena/internal/platform/database"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/middleware"
	"github/DoanCongPho/game-arena/migrations"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func helloWorld(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hello, world!\r\n")) }

func checkhealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
func main() {
	cfg := config.MustLoad()
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(database.RunMigrateCmd(cfg, migrations.FS, os.Args[2:]))
	}
	plat, err := platform.Build(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = plat.Close(context.Background()) }()

	r := mux.NewRouter()
	r.HandleFunc("/", helloWorld)
	r.HandleFunc("/health", checkhealth)

	// Protected routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.RequireAuth)

	// -- Auth --
	authRepo := auth.NewUserRepository(plat.DB)
	authSvc := auth.NewService(authRepo)
	authSvc.MountRoutes(r)
	auth.Init(cfg.App.SecretKey)
	middleware.CorsInit(cfg.App.AllowedOrigins...)

	// --internal--

	// --ielts_test--
	llmClient := llm.NewClient(cfg.App.OpenAIAPIKey, cfg.App.OpenAIModel)
	grader := ielts_test.NewOpenAIGrader(llmClient)

	testRepo := ielts_test.NewRepository(plat.DB)
	testSvc := ielts_test.NewService(testRepo, grader)

	// hot fix
	_ = testSvc

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           middleware.CORS(r),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failure: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server failure")
	}

}
