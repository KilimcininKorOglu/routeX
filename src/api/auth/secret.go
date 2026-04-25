package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"routex/constant"
)

const secretFileName = "auth_secret"

var (
	secretOnce  sync.Once
	secretValue []byte
	secretErr   error
)

func LoadAppSecret() ([]byte, error) {
	secretOnce.Do(func() {
		secretValue, secretErr = loadOrCreateSecret()
	})
	return secretValue, secretErr
}

func loadOrCreateSecret() ([]byte, error) {
	secretPath := filepath.Join(constant.AppStateDir, secretFileName)
	data, err := os.ReadFile(secretPath)
	if err == nil {
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return nil, errors.New("authentication key is empty")
		}
		decoded, err := base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("failed to decode authentication key: %w", err)
		}
		return decoded, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read authentication key: %w", err)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate authentication key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(secret)
	if err := os.MkdirAll(constant.AppStateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create authentication key directory: %w", err)
	}
	if err := os.WriteFile(secretPath, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to write authentication key: %w", err)
	}
	return secret, nil
}
