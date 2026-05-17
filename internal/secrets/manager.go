package secrets

import (
	"fmt"
	"os"
)

// SecretManager defines an interface for retrieving sensitive credentials.
type SecretManager interface {
	Get(key string) (string, error)
}

// EnvSecretManager retrieves secrets from environment variables.
type EnvSecretManager struct{}

// NewEnvSecretManager creates a new instance of EnvSecretManager.
func NewEnvSecretManager() *EnvSecretManager {
	return &EnvSecretManager{}
}

// Get retrieves a secret from an environment variable.
func (m *EnvSecretManager) Get(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("secret not found in environment: %s", key)
	}
	return val, nil
}
