package templates

import (
	"fmt"
	"strconv"
	"strings"
)

// Vars holds values available for {{VAR}} substitution in startup commands.
type Vars struct {
	RAM        int
	CPU        int
	Disk       int
	ServerPort int
	Extra      map[string]string
}

// Substitute replaces {{KEY}} placeholders. Unknown keys are left unchanged.
func Substitute(s string, v Vars) string {
	repl := map[string]string{
		"RAM":         strconv.Itoa(v.RAM),
		"CPU":         strconv.Itoa(v.CPU),
		"DISK":        strconv.Itoa(v.Disk),
		"SERVER_PORT": strconv.Itoa(v.ServerPort),
	}
	for k, val := range v.Extra {
		repl[strings.ToUpper(k)] = val
	}
	return replacePlaceholders(s, repl)
}

func replacePlaceholders(s string, repl map[string]string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			end := strings.Index(s[i+2:], "}}")
			if end >= 0 {
				key := strings.TrimSpace(s[i+2 : i+2+end])
				keyUpper := strings.ToUpper(key)
				if val, ok := repl[keyUpper]; ok {
					b.WriteString(val)
					i = i + 2 + end + 2
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ApplyStartup returns the template startup command with resource/env vars filled.
func ApplyStartup(def *Definition, memoryMB, cpuLimit, serverPort int) string {
	if def == nil {
		return ""
	}
	extra := map[string]string{}
	for k, v := range def.Environment {
		extra[k] = v
	}
	if serverPort <= 0 {
		serverPort = def.DefaultPort()
	}
	return Substitute(def.Startup.Command, Vars{
		RAM:        memoryMB,
		CPU:        cpuLimit,
		ServerPort: serverPort,
		Extra:      extra,
	})
}

// FormatPortList is a small helper for diagnostics/tests.
func FormatPortList(ports []int) string {
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, fmt.Sprintf("%d", p))
	}
	return strings.Join(parts, ",")
}
