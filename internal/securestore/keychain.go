package securestore

import (
	"strings"

	"github.com/zalando/go-keyring"
)

const service = "com.mux.desktop"

func Set(accountID, secret string) error {
	return keyring.Set(service, accountID, secret)
}

func Get(accountID string) (string, error) {
	return keyring.Get(service, accountID)
}

func Delete(accountID string) error {
	err := keyring.Delete(service, accountID)
	if err == nil || err == keyring.ErrNotFound || strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "could not be found") {
		return nil
	}
	return err
}
