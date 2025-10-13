package install

import (
	"strings"
	"wago-init/internal/fs"
)

// BuildContainerCommand constructs a single-line docker command from the stored configuration.
func BuildContainerCommand(cfg fs.EnvConfig) string {
	if cfg == nil {
		return ""
	}

	raw := fs.DecodeMultilineValue(cfg[fs.ContainerCommand])
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lines := strings.Split(raw, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}

	return strings.Join(parts, " ")
}
