package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/tinode/chat/server/logs"
)

// MessageEncryption handles encryption/decryption of message content at rest.
type MessageEncryption struct {
	enabled bool
	key     []byte
	gcm     cipher.AEAD
}

var msgEncryption *MessageEncryption

// InitMessageEncryption initializes the message encryption system.
// key should be a base64-encoded 32-byte (256-bit) AES key.
// If key is empty, encryption is disabled.
func InitMessageEncryption(keyBase64 string) error {
	if keyBase64 == "" {
		msgEncryption = &MessageEncryption{enabled: false}
		logs.Info.Println("Message encryption at rest: DISABLED")
		return nil
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return errors.New("invalid encryption key: " + err.Error())
	}

	if len(key) != 32 {
		return errors.New("encryption key must be 32 bytes (256-bit AES)")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	msgEncryption = &MessageEncryption{
		enabled: true,
		key:     key,
		gcm:     gcm,
	}

	logs.Info.Println("Message encryption at rest: ENABLED")
	return nil
}

// IsEncryptionEnabled returns true if message encryption is enabled.
func IsEncryptionEnabled() bool {
	return msgEncryption != nil && msgEncryption.enabled
}

// EncryptContent encrypts message content before storing to database.
// Returns the original content if encryption is disabled.
func EncryptContent(content any) (any, error) {
	if !IsEncryptionEnabled() {
		return content, nil
	}

	// Serialize content to JSON
	plaintext, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, msgEncryption.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt: nonce is prepended to ciphertext
	ciphertext := msgEncryption.gcm.Seal(nonce, nonce, plaintext, nil)

	// Return as base64 string with prefix to identify encrypted content
	return "ENC:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptContent decrypts message content after reading from database.
// Returns the original content if encryption is disabled or content is not encrypted.
func DecryptContent(content any) (any, error) {
	if !IsEncryptionEnabled() {
		return content, nil
	}

	// Check if content is an encrypted string
	str, ok := content.(string)
	if !ok {
		// Not a string, return as-is (might be old unencrypted content)
		return content, nil
	}

	// Check for encryption prefix
	if len(str) < 4 || str[:4] != "ENC:" {
		// Not encrypted, return as-is
		return content, nil
	}

	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(str[4:])
	if err != nil {
		return nil, errors.New("failed to decode encrypted content: " + err.Error())
	}

	// Extract nonce
	nonceSize := msgEncryption.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := msgEncryption.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("failed to decrypt content: " + err.Error())
	}

	// Deserialize JSON back to original type
	var result any
	if err := json.Unmarshal(plaintext, &result); err != nil {
		return nil, errors.New("failed to unmarshal decrypted content: " + err.Error())
	}

	return result, nil
}

// GenerateEncryptionKey generates a new random 256-bit encryption key.
// Returns the key as a base64-encoded string.
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
