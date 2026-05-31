package policyresolver

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func VerifyEd25519(publicKeyPath, artifactPath, signaturePath string) error {
	pubB64, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}

	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(pubB64)))
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: got=%d want=%d", len(pub), ed25519.PublicKeySize)
	}

	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("read artifact: %w", err)
	}

	sigB64, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("read signature: %w", err)
	}

	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size: got=%d want=%d", len(sig), ed25519.SignatureSize)
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), artifact, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}
