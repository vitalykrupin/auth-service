package main

import (
	"github.com/vitalykrupin/auth-service/cmd/auth/config"
	"go.uber.org/zap"
)

// applyMigrations is a stub when migrate build tag is not used.
func applyMigrations(conf *config.Config, logger *zap.SugaredLogger) error {
	// Only run when explicitly enabled and DSN provided; do nothing otherwise.
	if !conf.RunMigrations || conf.DBDSN == "" {
		return nil
	}
	logger.Infow("RUN_MIGRATIONS is true but migrate tag not enabled; skipping")
	return nil
}
