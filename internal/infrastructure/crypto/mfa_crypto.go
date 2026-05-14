package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// MFASecretCryptoService encrypts/decrypts user MFA shared secrets with AES-256 GCM.
type MFASecretCryptoService struct {
	aesKey []byte
}

func NewMFASecretCryptoService(aesKeyBase64 string) (*MFASecretCryptoService, error) {
	aesKey, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil || len(aesKey) != 32 {
		return nil, errors.New("MFA AES key must be 32 bytes base64-encoded")
	}

	return &MFASecretCryptoService{aesKey: aesKey}, nil
}

func (s *MFASecretCryptoService) EncryptString(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *MFASecretCryptoService) DecryptString(cipherB64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
