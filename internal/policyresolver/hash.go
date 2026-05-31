package policyresolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open artifact for hashing: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash artifact: %w", err)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func VerifySHA256(path string, expected string) error {
	actual, err := ComputeSHA256(path)
	if err != nil {
		return err
	}

	if strings.TrimSpace(strings.ToLower(actual)) != strings.TrimSpace(strings.ToLower(expected)) {
		return fmt.Errorf("artifact hash mismatch: expected=%s actual=%s", expected, actual)
	}

	return nil
}
