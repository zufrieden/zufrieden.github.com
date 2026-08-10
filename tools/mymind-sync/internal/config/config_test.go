package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromDotEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := `# mymind credentials
MYMIND_KEY_ID=abc123

export MYMIND_PRIVATE_KEY="sup3r\nsecret"
MYMIND_API_BASE_URL = https://api.mymind.com # override
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.KeyID != "abc123" {
		t.Errorf("KeyID = %q", cfg.KeyID)
	}
	if cfg.PrivateKey != "sup3r\nsecret" {
		t.Errorf("PrivateKey = %q", cfg.PrivateKey)
	}
	if cfg.BaseURL != "https://api.mymind.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestRealEnvironmentWinsOverDotEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("MYMIND_KEY_ID=from-file\nMYMIND_PRIVATE_KEY=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvKeyID, "from-secrets")
	t.Setenv(EnvPrivateKey, "from-secrets")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.KeyID != "from-secrets" || cfg.PrivateKey != "from-secrets" {
		t.Errorf("got %+v, want the environment values", cfg)
	}
}

func TestMissingCredentialsAreReported(t *testing.T) {
	t.Setenv(EnvKeyID, "")
	t.Setenv(EnvPrivateKey, "")

	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.env"))
	if err == nil {
		t.Fatal("expected an error when credentials are missing")
	}
	for _, want := range []string{EnvKeyID, EnvPrivateKey} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestMalformedLineIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("this is not an assignment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for a malformed .env line")
	}
}
