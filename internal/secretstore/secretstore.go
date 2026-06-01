package secretstore

import (
	"errors"
	"fmt"

	"github.com/danieljoos/wincred"
)

const targetPrefix = "EverestDataAgent/"

type Store struct{}

func New() *Store {
	return &Store{}
}

func (s *Store) Save(name string, value string) error {
	if name == "" {
		return fmt.Errorf("secret name is required")
	}
	if value == "" {
		return fmt.Errorf("secret value is required")
	}

	cred := wincred.NewGenericCredential(targetPrefix + name)
	cred.CredentialBlob = []byte(value)
	cred.Persist = wincred.PersistLocalMachine

	if err := cred.Write(); err != nil {
		return fmt.Errorf("write secret to Windows Credential Manager: %w", err)
	}

	return nil
}

func (s *Store) Load(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name is required")
	}

	cred, err := wincred.GetGenericCredential(targetPrefix + name)
	if err != nil {
		return "", fmt.Errorf("read secret from Windows Credential Manager: %w", err)
	}

	return string(cred.CredentialBlob), nil
}

func (s *Store) Exists(name string) bool {
	_, err := s.Load(name)
	return err == nil
}

func (s *Store) Delete(name string) error {
	if name == "" {
		return fmt.Errorf("secret name is required")
	}

	cred, err := wincred.GetGenericCredential(targetPrefix + name)
	if err != nil {
		return nil
	}

	if cred == nil {
		return nil
	}

	if err := cred.Delete(); err != nil && !errors.Is(err, wincred.ErrElementNotFound) {
		return fmt.Errorf("delete secret from Windows Credential Manager: %w", err)
	}

	return nil
}
