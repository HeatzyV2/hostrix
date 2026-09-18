package backups

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/agentclient"
	"github.com/hostrix/hostrix/api/internal/models"
	"github.com/hostrix/hostrix/api/internal/servers"
	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("backup not found")
	ErrForbidden = errors.New("forbidden")
)

func Create(ctx context.Context, db *gorm.DB, user *models.User, serverUUID, displayName string) (*models.Backup, error) {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return nil, err
	}
	if !servers.CanAccess(db, user, srv) {
		return nil, ErrForbidden
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "backup-" + time.Now().UTC().Format("20060102-150405")
	}
	if len(displayName) > 255 {
		return nil, fmt.Errorf("name too long")
	}

	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return nil, err
	}
	if node.Status == "OFFLINE" {
		return nil, fmt.Errorf("node is offline")
	}

	backupUUID := uuid.NewString()
	row := &models.Backup{
		UUID:      backupUUID,
		ServerID:  srv.ID,
		Name:      displayName,
		SizeBytes: 0,
		Status:    "PENDING",
	}
	if err := db.Create(row).Error; err != nil {
		return nil, err
	}

	_ = db.Model(row).Update("status", "RUNNING").Error
	row.Status = "RUNNING"

	client := agentclient.New(node.Address, node.Port, node.Token)
	info, err := client.CreateBackup(ctx, srv.ContainerName, backupUUID)
	if err != nil {
		_ = db.Model(row).Updates(map[string]any{"status": "FAILED"}).Error
		row.Status = "FAILED"
		return row, fmt.Errorf("create backup: %w", err)
	}

	updates := map[string]any{"status": "COMPLETED"}
	if info != nil && info.SizeBytes > 0 {
		updates["size_bytes"] = info.SizeBytes
		row.SizeBytes = info.SizeBytes
	}
	_ = db.Model(row).Updates(updates).Error
	row.Status = "COMPLETED"
	return row, nil
}

func ListForServer(db *gorm.DB, user *models.User, serverUUID string) ([]models.Backup, error) {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return nil, err
	}
	if !servers.CanAccess(db, user, srv) {
		return nil, ErrForbidden
	}
	var list []models.Backup
	err = db.Where("server_id = ?", srv.ID).Order("id desc").Find(&list).Error
	return list, err
}

func ListAccessible(db *gorm.DB, user *models.User) ([]models.Backup, error) {
	var list []models.Backup
	if user.IsAdmin {
		err := db.Order("id desc").Find(&list).Error
		return list, err
	}
	serverIDs, err := servers.AccessibleServerIDs(db, user)
	if err != nil {
		return nil, err
	}
	if len(serverIDs) == 0 {
		return []models.Backup{}, nil
	}
	err = db.Where("server_id IN ?", serverIDs).Order("id desc").Find(&list).Error
	return list, err
}

func GetByUUID(db *gorm.DB, backupUUID string) (*models.Backup, error) {
	var b models.Backup
	err := db.Where("uuid = ?", backupUUID).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &b, err
}

func GetForUser(db *gorm.DB, user *models.User, serverUUID, backupUUID string) (*models.Backup, *models.Server, error) {
	srv, err := servers.GetByUUID(db, serverUUID)
	if err != nil {
		return nil, nil, err
	}
	if !servers.CanAccess(db, user, srv) {
		return nil, nil, ErrForbidden
	}
	b, err := GetByUUID(db, backupUUID)
	if err != nil {
		return nil, nil, err
	}
	if b.ServerID != srv.ID {
		return nil, nil, ErrNotFound
	}
	return b, srv, nil
}

func Delete(ctx context.Context, db *gorm.DB, user *models.User, serverUUID, backupUUID string) error {
	b, srv, err := GetForUser(db, user, serverUUID, backupUUID)
	if err != nil {
		return err
	}
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return err
	}
	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.DeleteBackup(ctx, srv.ContainerName, b.UUID); err != nil {
		// Still remove DB row if Incus backup is already gone.
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return err
		}
	}
	return db.Delete(b).Error
}

func Restore(ctx context.Context, db *gorm.DB, user *models.User, serverUUID, backupUUID string) error {
	b, srv, err := GetForUser(db, user, serverUUID, backupUUID)
	if err != nil {
		return err
	}
	if b.Status != "COMPLETED" {
		return fmt.Errorf("backup is not ready")
	}
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return err
	}
	if node.Status == "OFFLINE" {
		return fmt.Errorf("node is offline")
	}

	_ = db.Model(b).Update("status", "RESTORING").Error
	_ = db.Model(srv).Update("status", "RESTORING").Error

	client := agentclient.New(node.Address, node.Port, node.Token)
	if err := client.RestoreBackup(ctx, srv.ContainerName, b.UUID); err != nil {
		_ = db.Model(b).Update("status", "COMPLETED").Error
		_ = db.Model(srv).Update("status", "ERROR").Error
		return err
	}
	_ = db.Model(b).Update("status", "COMPLETED").Error
	_ = db.Model(srv).Update("status", "RUNNING").Error
	return nil
}

func Download(ctx context.Context, db *gorm.DB, user *models.User, serverUUID, backupUUID string, w io.Writer) (int64, *models.Backup, error) {
	b, srv, err := GetForUser(db, user, serverUUID, backupUUID)
	if err != nil {
		return 0, nil, err
	}
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return 0, nil, err
	}
	client := agentclient.New(node.Address, node.Port, node.Token)
	n, err := client.DownloadBackup(ctx, srv.ContainerName, b.UUID, w)
	if err != nil {
		return n, b, err
	}
	if n > 0 && b.SizeBytes == 0 {
		_ = db.Model(b).Update("size_bytes", n).Error
		b.SizeBytes = n
	}
	return n, b, nil
}
