package users

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
)

func HashPassword(password string, cost int) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func EnsureBootstrapAdmin(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := HashPassword(cfg.BootstrapAdminPassword, cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("bootstrap password: %w", err)
	}

	admin := models.User{
		UUID:         uuid.NewString(),
		Username:     strings.TrimSpace(cfg.BootstrapAdminUsername),
		Email:        strings.ToLower(strings.TrimSpace(cfg.BootstrapAdminEmail)),
		PasswordHash: hash,
		IsAdmin:      true,
	}
	if admin.Username == "" || admin.Email == "" {
		return fmt.Errorf("bootstrap admin username/email required")
	}
	return db.Create(&admin).Error
}

func FindByLogin(db *gorm.DB, login string) (*models.User, error) {
	login = strings.TrimSpace(login)
	var user models.User
	err := db.Where("username = ? OR email = ?", login, strings.ToLower(login)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func List(db *gorm.DB) ([]models.User, error) {
	var list []models.User
	err := db.Order("id asc").Find(&list).Error
	return list, err
}
