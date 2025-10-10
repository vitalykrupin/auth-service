//go:build migrate

package main

import (
	"github.com/vitalykrupin/auth-service/cmd/auth/config"
	"go.uber.org/zap"

	m "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func applyMigrations(conf *config.Config, logger *zap.SugaredLogger) error {
	if !conf.RunMigrations || conf.DBDSN == "" {
		return nil
	}
	mig, err := m.New("file://"+conf.MigrationsPath, conf.DBDSN)
	if err != nil {
		return err
	}
	if err := mig.Up(); err != nil && err != m.ErrNoChange {
		return err
	}
	logger.Infow("migrations applied")
	return nil
}
