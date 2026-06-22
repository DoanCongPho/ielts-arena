package database

import (
	"database/sql"
	"errors"
	"fmt"
	config "github/DoanCongPho/game-arena/internal/platform/config"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // registers the "mysql" sql.Open driver
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate applies all pending up migrations against the database described by
// cfg. fsys is the migrations filesystem (e.g. migrations.FS).
func Migrate(cfg *config.DBConfig, fsys fs.FS) error {
	m, err := newMigrator(cfg, fsys)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// newMigrator opens a dedicated connection for migrations — separate from the
// app's runtime GORM pool. The connection carries multiStatements=true (see
// buildMigrationDSN); the runtime pool deliberately does not. The returned
// migrate.Migrate owns the connection: m.Close() closes it.
func newMigrator(cfg *config.DBConfig, fsys fs.FS) (*migrate.Migrate, error) {
	dsn, err := buildMigrationDSN(cfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration db: %w", err)
	}
	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("mysql driver: %w", err)
	}
	source, err := iofs.New(fsys, ".")
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("iofs source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return m, nil
}

// buildMigrationDSN is buildDSN plus multiStatements=true. golang-migrate's
// mysql driver runs each .sql file as a single Exec, so any migration file
// with more than one statement needs the multi-statement capability. The
// runtime pool (Open) omits it to keep multi-statement injection off the app
// query path.
func buildMigrationDSN(cfg *config.DBConfig) (string, error) {
	dsn, err := buildDSN(cfg)
	if err != nil {
		return "", err
	}
	return dsn + "&multiStatements=true", nil
}

// RunMigrateCmd is the subcommand entry point. Returns a process exit code.
//
// Usage:
//
//	./app migrate up
//	./app migrate down <N>
//	./app migrate status
//	./app migrate force <V>
//	./app migrate create <name>      (development only)
func RunMigrateCmd(cfg *config.Config, fsys fs.FS, args []string) int {
	if len(args) == 0 {
		printMigrateUsage()
		return 2
	}
	sub := args[0]

	if sub == "create" {
		if cfg.App.Env == "production" {
			fmt.Fprintln(os.Stderr, "migrate create is disabled in production")
			return 2
		}
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: ./app migrate create <name>")
			return 2
		}
		if err := createMigration(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	m, err := newMigrator(&cfg.DB, fsys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrator: %v\n", err)
		return 1
	}
	defer m.Close()

	switch sub {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("migrations: up to date")
		return 0
	case "down":
		n := 1
		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "down N must be int")
				return 2
			}
			n = v
		}
		if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("migrations: rolled back %d step(s)\n", n)
		return 0
	case "status":
		v, dirty, err := m.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("current version: %d dirty=%v\n", v, dirty)
		return 0
	case "force":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: ./app migrate force <V>")
			return 2
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "force V must be int")
			return 2
		}
		if err := m.Force(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("forced version: %d\n", v)
		return 0
	default:
		printMigrateUsage()
		return 2
	}
}

func printMigrateUsage() {
	fmt.Fprintln(os.Stderr, "usage: ./app migrate {up|down N|status|force V|create NAME}")
}

func createMigration(name string) error {
	clean := strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(name))
	timestamp := time.Now().Format("20060102150405")
	files, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}
	highest := 0
	for _, f := range files {
		parts := strings.SplitN(f.Name(), "_", 2)
		n, err := strconv.Atoi(parts[0])
		if err == nil && n > highest {
			highest = n
		}
	}
	num := fmt.Sprintf("%06d", highest+1)
	up := filepath.Join("migrations", num+"_"+clean+".up.sql")
	down := filepath.Join("migrations", num+"_"+clean+".down.sql")
	for _, p := range []string{up, down} {
		if err := os.WriteFile(p, []byte("-- "+filepath.Base(p)+" ("+timestamp+")\n"), 0o600); err != nil {
			return err
		}
		fmt.Println("created", p)
	}
	return nil
}
