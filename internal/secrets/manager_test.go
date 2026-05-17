package secrets

import (
	"os"
	"testing"
)

func TestEnvSecretManager(t *testing.T) {
	sm := NewEnvSecretManager()

	os.Setenv("TEST_KEY", "secret_value")
	defer os.Unsetenv("TEST_KEY")

	val, err := sm.Get("TEST_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != "secret_value" {
		t.Errorf("expected 'secret_value', got '%s'", val)
	}

	_, err = sm.Get("NON_EXISTENT")
	if err == nil {
		t.Error("expected error for non-existent secret, got nil")
	}
}
