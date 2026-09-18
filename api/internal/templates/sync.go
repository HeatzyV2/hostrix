package templates

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/models"
	"gorm.io/gorm"
)

// ResolveTemplatesDir picks HOSTRIX_TEMPLATES_DIR or common defaults.
func ResolveTemplatesDir(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{
		filepath.Clean("../templates"),
		"/opt/hostrix/templates",
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return filepath.Clean("../templates")
}

// SyncFromDisk upserts YAML templates into MariaDB by slug.
func SyncFromDisk(db *gorm.DB, dir string) (int, error) {
	dir = ResolveTemplatesDir(dir)
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("templates: directory %q not found, skipping sync", dir)
			return 0, nil
		}
		return 0, err
	}
	if !st.IsDir() {
		return 0, fmt.Errorf("templates path is not a directory: %s", dir)
	}

	defs, err := LoadDir(dir)
	if err != nil {
		return 0, err
	}

	n := 0
	for _, def := range defs {
		image := def.Image.ResolveImage()
		var existing models.ServerTemplate
		err := db.Where("slug = ?", def.Slug).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row := models.ServerTemplate{
				UUID:           uuid.NewString(),
				Name:           def.Name,
				Slug:           def.Slug,
				Description:    def.Description,
				Image:          image,
				StartupCommand: def.Startup.Command,
			}
			if err := db.Create(&row).Error; err != nil {
				return n, fmt.Errorf("create %s: %w", def.Slug, err)
			}
			n++
			continue
		}
		if err != nil {
			return n, err
		}
		if err := db.Model(&existing).Updates(map[string]any{
			"name":            def.Name,
			"description":     def.Description,
			"image":           image,
			"startup_command": def.Startup.Command,
		}).Error; err != nil {
			return n, fmt.Errorf("update %s: %w", def.Slug, err)
		}
		n++
	}

	// Keep a base ubuntu fallback if YAML set did not include it.
	var count int64
	if err := db.Model(&models.ServerTemplate{}).Where("slug = ?", "ubuntu").Count(&count).Error; err != nil {
		return n, err
	}
	if count == 0 {
		if err := db.Create(&models.ServerTemplate{
			UUID:           uuid.NewString(),
			Name:           "Ubuntu",
			Slug:           "ubuntu",
			Description:    "Base Ubuntu 24.04 container",
			Image:          "ubuntu/24.04",
			StartupCommand: "",
		}).Error; err != nil {
			return n, err
		}
		n++
	}

	log.Printf("templates: synced %d definition(s) from %s", n, dir)
	return n, nil
}
