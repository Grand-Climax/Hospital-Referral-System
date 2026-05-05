package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
)

type PatientCryptoService struct {
	aesKey  []byte
	hmacKey []byte
}

func NewPatientCryptoService(aesKeyBase64, hmacKeyBase64 string) (*PatientCryptoService, error) {
	aesKey, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil || len(aesKey) != 32 {
		return nil, errors.New("AES key must be 32 bytes base64-encoded")
	}
	hmacKey, err := base64.StdEncoding.DecodeString(hmacKeyBase64)
	if err != nil || len(hmacKey) < 32 {
		return nil, errors.New("HMAC key must be at least 32 bytes base64-encoded")
	}
	return &PatientCryptoService{aesKey: aesKey, hmacKey: hmacKey}, nil
}

// Encrypt returns base64(iv || ciphertext || tag)
func (s *PatientCryptoService) Encrypt(plaintext []byte) (string, error) {
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
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt expects base64(iv || ciphertext || tag)
func (s *PatientCryptoService) Decrypt(cipherB64 string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GenerateHMAC returns hex-encoded HMAC-SHA256 of the value
func (s *PatientCryptoService) GenerateHMAC(value string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// NormalizePhone converts Ethiopian phone numbers to +2519XXXXXXXX or +2517XXXXXXXX.
// Returns error if the number cannot be normalised.
func NormalizePhone(raw string) (string, error) {
	// Remove spaces, dashes, parentheses
	cleaned := regexp.MustCompile(`[^\d\+]`).ReplaceAllString(raw, "")
	// Handle common prefixes
	if strings.HasPrefix(cleaned, "0") && len(cleaned) == 10 {
		switch cleaned[1] {
		case '9':
			cleaned = "+251" + cleaned[1:] // 09... -> +2519...
		case '7':
			cleaned = "+251" + cleaned[1:] // 07... -> +2517...
		default:
			return "", errors.New("invalid Ethiopian mobile prefix")
		}
	} else if strings.HasPrefix(cleaned, "00251") && len(cleaned) == 14 {
		cleaned = "+" + cleaned[2:] // 002519... -> +2519...
	} else if strings.HasPrefix(cleaned, "+2510") && len(cleaned) == 13 {
		cleaned = "+251" + cleaned[5:] // +25109... -> +2519...
	} else if strings.HasPrefix(cleaned, "+251") && len(cleaned) == 13 {
		// already correct
	} else {
		return "", errors.New("unrecognized phone number format")
    }
    
	// Final check: must start with +251 and be 13 chars
	if !strings.HasPrefix(cleaned, "+251") || len(cleaned) != 13 {
		return "", errors.New("invalid phone number length after normalization")
	}
	return cleaned, nil
}
