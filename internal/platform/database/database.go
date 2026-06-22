package database

import (
	"database/sql"
	"errors"
	"fmt"
	config "github/DoanCongPho/game-arena/internal/platform/config"
	"log"
	"strings"
	"time"
)

func Open(cfg *config.DBConfig) (*sql.DB, error) {
	if cfg == nil {
		return nil, errors.New("DBConfig is nil")
	}
	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	fmt.Println("Successfully connected to the MySQL database!")
	return db, nil
}

func buildDSN(cfg *config.DBConfig) (string, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return "", errors.New("DBConfig.Host is empty")
	}
	if cfg.Port <= 0 {
		return "", fmt.Errorf("DBConfig.Port invalid: %d", cfg.Port)
	}
	user := strings.TrimSpace(cfg.User)
	if user == "" {
		return "", errors.New("DBConfig.User is empty")
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return "", errors.New("DBConfig.Name is empty")
	}
	// %27 = "'", %2B = "+", %3A = ":" → time_zone='+00:00'
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC&time_zone=%%27%%2B00%%3A00%%27&charset=utf8mb4",
		user, cfg.Password, host, cfg.Port, name,
	), nil
}
