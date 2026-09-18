package templates

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/models"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("template not found")
	slugRe      = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type CreateInput struct {
	Name           string
	Slug           string
	Description    string
	Image          string
	StartupCommand string
}

type UpdateInput struct {
	Name           *string
	Slug           *string
	Description    *string
	Image          *string
	StartupCommand *string
}

func List(db *gorm.DB) ([]models.ServerTemplate, error) {
	var list []models.ServerTemplate
	err := db.Order("name asc").Find(&list).Error
	return list, err
}

func GetByUUID(db *gorm.DB, id string) (*models.ServerTemplate, error) {
	var t models.ServerTemplate
	err := db.Where("uuid = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &t, err
}

func GetBySlug(db *gorm.DB, slug string) (*models.ServerTemplate, error) {
	var t models.ServerTemplate
	err := db.Where("slug = ?", strings.TrimSpace(slug)).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &t, err
}

func Create(db *gorm.DB, in CreateInput) (*models.ServerTemplate, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Description = strings.TrimSpace(in.Description)
	in.Image = strings.TrimSpace(in.Image)
	in.StartupCommand = strings.TrimSpace(in.StartupCommand)
	if in.Name == "" || in.Slug == "" || in.Image == "" {
		return nil, fmt.Errorf("name, slug and image are required")
	}
	if !slugRe.MatchString(in.Slug) {
		return nil, fmt.Errorf("invalid slug")
	}
	t := &models.ServerTemplate{
		UUID:           uuid.NewString(),
		Name:           in.Name,
		Slug:           in.Slug,
		Description:    in.Description,
		Image:          in.Image,
		StartupCommand: in.StartupCommand,
	}
	if err := db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func Update(db *gorm.DB, id string, in UpdateInput) (*models.ServerTemplate, error) {
	t, err := GetByUUID(db, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		updates["name"] = name
	}
	if in.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*in.Slug))
		if !slugRe.MatchString(slug) {
			return nil, fmt.Errorf("invalid slug")
		}
		updates["slug"] = slug
	}
	if in.Description != nil {
		updates["description"] = strings.TrimSpace(*in.Description)
	}
	if in.Image != nil {
		image := strings.TrimSpace(*in.Image)
		if image == "" {
			return nil, fmt.Errorf("image cannot be empty")
		}
		updates["image"] = image
	}
	if in.StartupCommand != nil {
		updates["startup_command"] = strings.TrimSpace(*in.StartupCommand)
	}
	if len(updates) == 0 {
		return t, nil
	}
	if err := db.Model(t).Updates(updates).Error; err != nil {
		return nil, err
	}
	return GetByUUID(db, id)
}

func Delete(db *gorm.DB, id string) error {
	t, err := GetByUUID(db, id)
	if err != nil {
		return err
	}
	var count int64
	if err := db.Model(&models.Server{}).Where("template_id = ?", t.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("template is in use by %d server(s)", count)
	}
	return db.Delete(t).Error
}

// ResolveRef looks up a template by numeric ID, UUID, or slug.
func ResolveRef(db *gorm.DB, templateID uint64, templateUUID, templateSlug string) (*models.ServerTemplate, error) {
	if templateUUID != "" {
		return GetByUUID(db, templateUUID)
	}
	if templateSlug != "" {
		return GetBySlug(db, templateSlug)
	}
	if templateID != 0 {
		var t models.ServerTemplate
		err := db.First(&t, templateID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return &t, err
	}
	return nil, nil
}
