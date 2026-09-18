package database

import (
	"fmt"
	"time"

	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/templates"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Setting{},
		&models.Node{},
		&models.ServerTemplate{},
		&models.Server{},
		&models.Allocation{},
		&models.Backup{},
		&models.ServerPermission{},
	); err != nil {
		return err
	}
	return nil
}

// SeedTemplates syncs YAML definitions from disk into MariaDB.
func SeedTemplates(db *gorm.DB, templatesDir string) error {
	_, err := templates.SyncFromDisk(db, templatesDir)
	return err
}
