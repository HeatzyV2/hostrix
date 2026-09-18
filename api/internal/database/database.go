package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/models"

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
	return seedTemplates(db)
}

func seedTemplates(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.ServerTemplate{}).Where("slug = ?", "ubuntu").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Create(&models.ServerTemplate{
		UUID:           uuid.NewString(),
		Name:           "Ubuntu",
		Slug:           "ubuntu",
		Description:    "Base Ubuntu 24.04 container (Phase 2)",
		Image:          "ubuntu/24.04",
		StartupCommand: "",
	}).Error
}
