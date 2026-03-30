package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	SaltSize  = 32
	NonceSize = 12
	KeySize   = 32 // AES-256

	// Argon2id parameters (OWASP recommended)
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4

	// Export file format magic bytes
	MagicHeader  = "RECL"
	FormatVersion = byte(1)
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: wrong password or corrupted data")
	ErrInvalidFormat    = errors.New("invalid export file format")
)

// GenerateSalt returns a cryptographically random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// DeriveKey uses Argon2id to derive a 256-bit key from a password and salt.
func DeriveKey(password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("password must not be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt must not be empty")
	}
	if len(salt) < 16 {
		return nil, fmt.Errorf("salt too short: %d bytes (minimum 16)", len(salt))
	}
	return argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, KeySize), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the given key.
// Returns: nonce + ciphertext (nonce is prepended).
func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data produced by Encrypt using AES-256-GCM.
// Input format: nonce + ciphertext.
func Decrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// PackExport assembles the export file format:
// magic (4 bytes) + version (1 byte) + salt (32 bytes) + nonce+ciphertext
func PackExport(salt, encryptedData []byte) []byte {
	header := make([]byte, 0, len(MagicHeader)+1+len(salt)+len(encryptedData))
	header = append(header, []byte(MagicHeader)...)
	header = append(header, FormatVersion)
	header = append(header, salt...)
	header = append(header, encryptedData...)
	return header
}

// UnpackExport parses the export file format and returns salt + encrypted data.
func UnpackExport(data []byte) (salt, encryptedData []byte, err error) {
	headerLen := len(MagicHeader) + 1 + SaltSize
	if len(data) < headerLen {
		return nil, nil, ErrInvalidFormat
	}

	if string(data[:len(MagicHeader)]) != MagicHeader {
		return nil, nil, ErrInvalidFormat
	}

	if data[len(MagicHeader)] != FormatVersion {
		return nil, nil, fmt.Errorf("unsupported export format version: %d", data[len(MagicHeader)])
	}

	saltStart := len(MagicHeader) + 1
	salt = data[saltStart : saltStart+SaltSize]
	encryptedData = data[saltStart+SaltSize:]

	if len(encryptedData) == 0 {
		return nil, nil, ErrInvalidFormat
	}

	return salt, encryptedData, nil
}

// ReadPassword reads a password from the terminal without echoing.
// Falls back to standard input if not a terminal.
func ReadPassword(prompt string) (string, error) {
	return readPasswordFromTerminal(prompt)
}

// ConfirmPassword reads a password twice and verifies they match.
// Warns if password is shorter than 8 characters.
func ConfirmPassword(prompt string) (string, error) {
	pass1, err := ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	if pass1 == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	if len(pass1) < 8 {
		fmt.Println("⚠  Warning: password is shorter than 8 characters, this is insecure")
	}

	pass2, err := ReadPassword("Confirm password: ")
	if err != nil {
		return "", err
	}

	if pass1 != pass2 {
		return "", fmt.Errorf("passwords do not match")
	}

	return pass1, nil
}
