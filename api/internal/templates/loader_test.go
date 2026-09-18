package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveImageFromMapping(t *testing.T) {
	def, err := LoadFile(writeTempYAML(t, `
name: Node.js
slug: nodejs
description: test
image:
  os: ubuntu
  release: "24.04"
startup:
  command: node index.js
ports:
  - 3000
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Image.ResolveImage(); got != "ubuntu/24.04" {
		t.Fatalf("image=%q", got)
	}
}

func TestResolveImageFromString(t *testing.T) {
	def, err := LoadFile(writeTempYAML(t, `
name: Custom
slug: custom
image: debian/12
startup:
  command: bash
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := def.Image.ResolveImage(); got != "debian/12" {
		t.Fatalf("image=%q", got)
	}
}

func TestLoadDir(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "minecraft-java")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(subdir, "template.yaml")
	content := `
name: Minecraft Java
slug: minecraft-java
description: mc
image:
  os: ubuntu
  release: "24.04"
startup:
  command: java -Xmx{{RAM}}M -jar server.jar
ports:
  - 25565
environment:
  SERVER_JAR: server.jar
variables:
  - RAM
  - SERVER_PORT
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	defs, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("len=%d", len(defs))
	}
	if defs[0].Slug != "minecraft-java" {
		t.Fatalf("slug=%s", defs[0].Slug)
	}
	if defs[0].DefaultPort() != 25565 {
		t.Fatalf("port=%d", defs[0].DefaultPort())
	}
}

func writeTempYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
