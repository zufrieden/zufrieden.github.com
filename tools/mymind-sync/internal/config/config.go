// Package config resolves credentials from the process environment, optionally
// seeded from a .env file. Real environment variables always win, so the same
// binary works locally (.env) and in CI (repo secrets).
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	EnvKeyID      = "MYMIND_KEY_ID"
	EnvPrivateKey = "MYMIND_PRIVATE_KEY"
	EnvBaseURL    = "MYMIND_API_BASE_URL"
)

type Config struct {
	KeyID      string
	PrivateKey string
	BaseURL    string
}

// Load reads envPath (when it exists) into the environment without clobbering
// existing variables, then pulls the config out of the environment.
func Load(envPath string) (Config, error) {
	if envPath != "" {
		if err := loadDotEnv(envPath); err != nil {
			return Config{}, err
		}
	}

	cfg := Config{
		KeyID:      strings.TrimSpace(os.Getenv(EnvKeyID)),
		PrivateKey: strings.TrimSpace(os.Getenv(EnvPrivateKey)),
		BaseURL:    strings.TrimSpace(os.Getenv(EnvBaseURL)),
	}

	var missing []string
	if cfg.KeyID == "" {
		missing = append(missing, EnvKeyID)
	}
	if cfg.PrivateKey == "" {
		missing = append(missing, EnvPrivateKey)
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("missing credentials: %s (set them in %s or the environment)",
			strings.Join(missing, ", "), envPath)
	}

	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .env is optional; CI uses real env vars.
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("%s:%d: expected KEY=value", path, lineNo)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNo)
		}
		if _, alreadySet := os.LookupEnv(key); alreadySet {
			continue
		}
		if err := os.Setenv(key, unquote(strings.TrimSpace(value))); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func unquote(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			inner := value[1 : len(value)-1]
			if first == '"' {
				inner = strings.NewReplacer(`\n`, "\n", `\"`, `"`, `\\`, `\`).Replace(inner)
			}
			return inner
		}
	}
	// Strip a trailing inline comment on unquoted values.
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}
