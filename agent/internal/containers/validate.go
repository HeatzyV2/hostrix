package containers

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var containerNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

func ValidateContainerName(name string) error {
	name = strings.TrimSpace(name)
	if !containerNameRe.MatchString(name) {
		return fmt.Errorf("invalid container name")
	}
	if strings.Contains(name, "--") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("invalid container name")
	}
	return nil
}

func ValidateCreateRequest(req CreateRequest) error {
	if err := ValidateContainerName(req.Name); err != nil {
		return err
	}
	if strings.TrimSpace(req.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if req.MemoryMB < 64 || req.MemoryMB > 1024*1024 {
		return fmt.Errorf("memory must be between 64 and 1048576 MB")
	}
	if req.CPULimit < 10 || req.CPULimit > 12800 {
		return fmt.Errorf("cpu must be between 10 and 12800 percent")
	}
	if req.DiskMB < 1024 || req.DiskMB > 10*1024*1024 {
		return fmt.Errorf("disk must be between 1024 and 10485760 MB")
	}
	return nil
}

// SanitizeContainerPath blocks path traversal and absolute host paths.
// Paths are always treated as absolute inside the container root.
func SanitizeContainerPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.Contains(p, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(p, "/"))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid path")
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	// Reject attempts to escape via .. after clean (should already be gone).
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal denied")
	}
	return cleaned, nil
}
