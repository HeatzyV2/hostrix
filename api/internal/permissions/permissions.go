package permissions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/servers"
	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("permission not found")
	ErrForbidden = errors.New("forbidden")
)

type GrantInput struct {
	UserUUID   string
	Username   string
	CanStart   bool
	CanStop    bool
	CanFiles   bool
	CanConsole bool
}

func ListForServer(db *gorm.DB, actor *models.User, serverUUID string) ([]models.ServerPermission, error) {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return nil, err
	}
	if !servers.CanManagePermissions(actor, srv) {
		return nil, ErrForbidden
	}
	var list []models.ServerPermission
	err = db.Where("server_id = ?", srv.ID).Order("id asc").Find(&list).Error
	return list, err
}

func Grant(db *gorm.DB, actor *models.User, serverUUID string, in GrantInput) (*models.ServerPermission, error) {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return nil, err
	}
	if !servers.CanManagePermissions(actor, srv) {
		return nil, ErrForbidden
	}
	in.UserUUID = strings.TrimSpace(in.UserUUID)
	in.Username = strings.TrimSpace(in.Username)

	var target models.User
	switch {
	case in.UserUUID != "":
		if err := db.Where("uuid = ?", in.UserUUID).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("user not found")
			}
			return nil, err
		}
	case in.Username != "":
		if err := db.Where("username = ? OR email = ?", in.Username, strings.ToLower(in.Username)).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("user not found")
			}
			return nil, err
		}
	default:
		return nil, fmt.Errorf("user_uuid or username is required")
	}
	if target.ID == srv.OwnerID {
		return nil, fmt.Errorf("owner already has full access")
	}
	if !in.CanStart && !in.CanStop && !in.CanFiles && !in.CanConsole {
		return nil, fmt.Errorf("at least one permission flag is required")
	}

	var perm models.ServerPermission
	err = db.Where("server_id = ? AND user_id = ?", srv.ID, target.ID).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		perm = models.ServerPermission{
			ServerID:   srv.ID,
			UserID:     target.ID,
			CanStart:   in.CanStart,
			CanStop:    in.CanStop,
			CanFiles:   in.CanFiles,
			CanConsole: in.CanConsole,
		}
		if err := db.Create(&perm).Error; err != nil {
			return nil, err
		}
		return &perm, nil
	}
	if err != nil {
		return nil, err
	}
	perm.CanStart = in.CanStart
	perm.CanStop = in.CanStop
	perm.CanFiles = in.CanFiles
	perm.CanConsole = in.CanConsole
	if err := db.Save(&perm).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}

func Revoke(db *gorm.DB, actor *models.User, serverUUID, userUUID string) error {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return err
	}
	if !servers.CanManagePermissions(actor, srv) {
		return ErrForbidden
	}
	var target models.User
	if err := db.Where("uuid = ?", userUUID).First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	res := db.Where("server_id = ? AND user_id = ?", srv.ID, target.ID).Delete(&models.ServerPermission{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// EnrichWithUser returns permission maps with user public fields for API responses.
func EnrichWithUser(db *gorm.DB, perms []models.ServerPermission) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(perms))
	for i := range perms {
		var u models.User
		if err := db.First(&u, perms[i].UserID).Error; err != nil {
			continue
		}
		out = append(out, map[string]any{
			"user_uuid":   u.UUID,
			"username":    u.Username,
			"email":       u.Email,
			"can_start":   perms[i].CanStart,
			"can_stop":    perms[i].CanStop,
			"can_files":   perms[i].CanFiles,
			"can_console": perms[i].CanConsole,
			"created_at":  perms[i].CreatedAt,
			"updated_at":  perms[i].UpdatedAt,
		})
	}
	return out, nil
}
