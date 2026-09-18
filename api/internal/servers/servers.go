package servers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hostrix/hostrix/api/internal/agentclient"
	"github.com/hostrix/hostrix/api/internal/models"
	tpl "github.com/hostrix/hostrix/api/internal/templates"
	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("server not found")
	ErrForbidden = errors.New("forbidden")
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 _.-]{1,62}$`)

type CreateInput struct {
	Name         string
	NodeUUID     string
	TemplateID   uint64 // legacy numeric id
	TemplateUUID string
	TemplateSlug string
	Image        string
	MemoryMB     int
	CPULimit     int
	DiskMB       int
	OwnerID      uint64
}

func Create(ctx context.Context, db *gorm.DB, in CreateInput) (*models.Server, error) {
	in.Name = strings.TrimSpace(in.Name)
	if !nameRe.MatchString(in.Name) {
		return nil, fmt.Errorf("invalid server name")
	}
	if in.MemoryMB <= 0 {
		in.MemoryMB = 1024
	}
	if in.CPULimit <= 0 {
		in.CPULimit = 100
	}
	if in.DiskMB <= 0 {
		in.DiskMB = 10240
	}
	if in.MemoryMB < 64 || in.CPULimit < 10 || in.DiskMB < 1024 {
		return nil, fmt.Errorf("invalid resource limits")
	}

	var node models.Node
	if err := db.Where("uuid = ?", in.NodeUUID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("node not found")
		}
		return nil, err
	}

	tmpl, err := tpl.ResolveRef(db, in.TemplateID, in.TemplateUUID, in.TemplateSlug)
	if err != nil {
		if errors.Is(err, tpl.ErrNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, err
	}

	var templateID uint64
	if tmpl != nil {
		templateID = tmpl.ID
		if in.Image == "" && tmpl.Image != "" {
			in.Image = tmpl.Image
		}
	} else {
		var fallback models.ServerTemplate
		if err := db.Where("slug = ?", "ubuntu").First(&fallback).Error; err == nil {
			templateID = fallback.ID
			if in.Image == "" {
				in.Image = fallback.Image
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			fallback = models.ServerTemplate{
				UUID:           uuid.NewString(),
				Name:           "Ubuntu",
				Slug:           "ubuntu",
				Description:    "Base Ubuntu container",
				Image:          "ubuntu/24.04",
				StartupCommand: "",
			}
			if err := db.Create(&fallback).Error; err != nil {
				return nil, err
			}
			templateID = fallback.ID
			if in.Image == "" {
				in.Image = fallback.Image
			}
		} else {
			return nil, err
		}
	}

	if in.Image == "" {
		in.Image = "ubuntu/24.04"
	}

	serverUUID := uuid.NewString()
	containerName := "hx-" + strings.ReplaceAll(serverUUID[:8], "-", "") + "-" + slugify(in.Name)

	srv := &models.Server{
		UUID:          serverUUID,
		Name:          in.Name,
		NodeID:        node.ID,
		TemplateID:    templateID,
		OwnerID:       in.OwnerID,
		ContainerName: containerName,
		MemoryMB:      in.MemoryMB,
		CPULimit:      in.CPULimit,
		DiskMB:        in.DiskMB,
		Status:        "INSTALLING",
	}
	if err := db.Create(srv).Error; err != nil {
		return nil, err
	}

	client := agentclient.New(node.Address, node.Port, node.Token)
	err = client.CreateContainer(ctx, agentclient.CreateContainerRequest{
		Name:     containerName,
		Image:    in.Image,
		MemoryMB: in.MemoryMB,
		CPULimit: in.CPULimit,
		DiskMB:   in.DiskMB,
	})
	if err != nil {
		_ = db.Model(srv).Update("status", "ERROR").Error
		return srv, fmt.Errorf("create container: %w", err)
	}

	if err := client.StartContainer(ctx, containerName); err != nil {
		_ = db.Model(srv).Update("status", "ERROR").Error
		return srv, fmt.Errorf("start container: %w", err)
	}
	_ = db.Model(srv).Update("status", "RUNNING").Error
	srv.Status = "RUNNING"
	return srv, nil
}

func ListForUser(db *gorm.DB, user *models.User) ([]models.Server, error) {
	var list []models.Server
	q := db.Order("id desc")
	if !user.IsAdmin {
		q = q.Where("owner_id = ?", user.ID)
	}
	err := q.Find(&list).Error
	return list, err
}

func GetByUUID(db *gorm.DB, id string) (*models.Server, error) {
	var srv models.Server
	err := db.Where("uuid = ?", id).First(&srv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &srv, err
}

func CanAccess(user *models.User, srv *models.Server) bool {
	return user.IsAdmin || srv.OwnerID == user.ID
}

func Delete(ctx context.Context, db *gorm.DB, user *models.User, id string) error {
	srv, err := GetByUUID(db, id)
	if err != nil {
		return err
	}
	if !CanAccess(user, srv) {
		return ErrForbidden
	}
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return err
	}
	client := agentclient.New(node.Address, node.Port, node.Token)
	_ = client.StopContainer(ctx, srv.ContainerName)
	if err := client.DeleteContainer(ctx, srv.ContainerName); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return err
		}
	}
	return db.Delete(srv).Error
}

func Power(ctx context.Context, db *gorm.DB, user *models.User, id, action string) error {
	srv, err := GetByUUID(db, id)
	if err != nil {
		return err
	}
	if !CanAccess(user, srv) {
		return ErrForbidden
	}
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return err
	}
	client := agentclient.New(node.Address, node.Port, node.Token)

	var pending string
	switch action {
	case "start":
		pending = "STARTING"
	case "stop":
		pending = "STOPPING"
	case "restart":
		pending = "RESTARTING"
	case "kill":
		pending = "STOPPING"
	default:
		return fmt.Errorf("invalid action")
	}
	_ = db.Model(srv).Update("status", pending).Error

	var opErr error
	switch action {
	case "start":
		opErr = client.StartContainer(ctx, srv.ContainerName)
	case "stop":
		opErr = client.StopContainer(ctx, srv.ContainerName)
	case "restart":
		opErr = client.RestartContainer(ctx, srv.ContainerName)
	case "kill":
		opErr = client.KillContainer(ctx, srv.ContainerName)
	}
	if opErr != nil {
		_ = db.Model(srv).Update("status", "ERROR").Error
		return opErr
	}

	status, err := client.GetStatus(ctx, srv.ContainerName)
	if err != nil {
		_ = db.Model(srv).Update("status", "UNKNOWN").Error
		return nil
	}
	_ = db.Model(srv).Update("status", status).Error
	return nil
}

func SyncStatus(ctx context.Context, db *gorm.DB, srv *models.Server) (string, error) {
	var node models.Node
	if err := db.First(&node, srv.NodeID).Error; err != nil {
		return "", err
	}
	client := agentclient.New(node.Address, node.Port, node.Token)
	status, err := client.GetStatus(ctx, srv.ContainerName)
	if err != nil {
		return "", err
	}
	_ = db.Model(srv).Update("status", status).Error
	return status, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 24 {
		out = out[:24]
	}
	if out == "" {
		out = "srv"
	}
	return out
}
