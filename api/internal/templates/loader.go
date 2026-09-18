package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Definition is a parsed YAML service template.
type Definition struct {
	Name        string            `yaml:"name" json:"name"`
	Slug        string            `yaml:"slug" json:"slug"`
	Description string            `yaml:"description" json:"description"`
	Image       ImageSpec         `yaml:"image" json:"image"`
	Startup     StartupSpec       `yaml:"startup" json:"startup"`
	Ports       []int             `yaml:"ports" json:"ports"`
	Environment map[string]string `yaml:"environment" json:"environment"`
	Variables   []string          `yaml:"variables" json:"variables"`
}

type StartupSpec struct {
	Command string `yaml:"command" json:"command"`
}

// ImageSpec accepts either a string ("ubuntu/24.04") or {os, release}.
type ImageSpec struct {
	OS      string `yaml:"os" json:"os,omitempty"`
	Release string `yaml:"release" json:"release,omitempty"`
	Alias   string `yaml:"-" json:"alias,omitempty"`
}

func (i *ImageSpec) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	switch value.Kind {
	case yaml.ScalarNode:
		i.Alias = strings.TrimSpace(value.Value)
		return nil
	case yaml.MappingNode:
		var raw struct {
			OS      string `yaml:"os"`
			Release string `yaml:"release"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		i.OS = strings.TrimSpace(raw.OS)
		i.Release = strings.TrimSpace(raw.Release)
		return nil
	default:
		return fmt.Errorf("image: expected string or mapping")
	}
}

// ResolveImage returns an Incus image alias such as "ubuntu/24.04".
func (i ImageSpec) ResolveImage() string {
	if alias := strings.TrimSpace(i.Alias); alias != "" {
		return alias
	}
	osName := strings.TrimSpace(i.OS)
	release := strings.TrimSpace(i.Release)
	if osName == "" {
		return ""
	}
	if release == "" {
		return osName
	}
	return osName + "/" + release
}

// DefaultPort returns the first declared port, or 0.
func (d *Definition) DefaultPort() int {
	if len(d.Ports) == 0 {
		return 0
	}
	return d.Ports[0]
}

// LoadFile parses a single template.yaml file.
func LoadFile(path string) (*Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	def.Name = strings.TrimSpace(def.Name)
	def.Slug = strings.TrimSpace(def.Slug)
	def.Description = strings.TrimSpace(def.Description)
	def.Startup.Command = strings.TrimSpace(def.Startup.Command)
	if def.Name == "" || def.Slug == "" {
		return nil, fmt.Errorf("%s: name and slug are required", path)
	}
	if def.Image.ResolveImage() == "" {
		return nil, fmt.Errorf("%s: image is required", path)
	}
	if def.Environment == nil {
		def.Environment = map[string]string{}
	}
	return &def, nil
}

// LoadDir walks dir for */template.yaml and returns parsed definitions.
func LoadDir(dir string) ([]*Definition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*Definition
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "template.yaml")
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		def, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}
