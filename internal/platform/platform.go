package platform

import (
	"context"
	"database/sql"
	"github/DoanCongPho/game-arena/internal/platform/config"
	"github/DoanCongPho/game-arena/internal/platform/database"
)

type Platform struct {
	Cfg *config.Config
	// Log *logrus.Logger
	DB *sql.DB
	// Audit       audit.Recorder
	// Settings    settings.Service
	// Auth        auth.Service
	// Users       auth.Repository
	// LoginEvents auth.LoginEventRepository
	// I18n        *i18n.Bundle
	// Mailer      mailer.Mailer
}

func Build(cfg *config.Config) (*Platform, error) {
	db, err := database.Open(&cfg.DB)

	if err != nil {
		return nil, err
	}

	return &Platform{
		Cfg: cfg,
		DB:  db,
	}, nil
}

func (p *Platform) Close(_ context.Context) error {
	if p == nil {
		return nil
	}
	// sentry.Flush(2 * time.Second)
	if p.DB == nil {
		return nil
	}
	return p.DB.Close()
}
