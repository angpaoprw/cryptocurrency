package controller

import (
	"fmt"

	"github.com/angpaoprw/cryptocurrency/api"
	"github.com/angpaoprw/cryptocurrency/crypto"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"go.uber.org/zap"
)

type Controller struct {
	sql                *db.Store
	notificationClient *api.NotificationClient
	encryptionKey      []byte
}

func NewController(sql *db.Store, notificationClient *api.NotificationClient, encryptionKey []byte) *Controller {
	return &Controller{
		sql:                sql,
		notificationClient: notificationClient,
		encryptionKey:      encryptionKey,
	}
}

// encryptPrivateKey encrypts a private key for storage
func (s *Controller) encryptPrivateKey(privateKey string) (string, error) {
	encrypted, err := crypto.Encrypt(privateKey, s.encryptionKey)
	if err != nil {
		logger.Error("Failed to encrypt private key", zap.Error(err))
		return "", fmt.Errorf("encryption failed: %w", err)
	}
	return encrypted, nil
}

// decryptPrivateKey decrypts a private key from storage
func (s *Controller) decryptPrivateKey(encryptedKey string) (string, error) {
	decrypted, err := crypto.Decrypt(encryptedKey, s.encryptionKey)
	if err != nil {
		logger.Error("Failed to decrypt private key", zap.Error(err))
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	return decrypted, nil
}
