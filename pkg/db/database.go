package db

import (
	"log/slog"
	"os"

	"routePlanner/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(dsn string, logger *slog.Logger) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Place{},
	)

	if err != nil {
		logger.Error("Failed to auto-migrate database", "error", err)
		os.Exit(1)
	}

	logger.Info("Database connection established and migrations complete.")
	return db
}
