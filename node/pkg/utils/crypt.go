package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"

	"golang.org/x/crypto/scrypt"
)

func EncryptText(textToEncrypt string) (string, error) {
	password := os.Getenv("ENCRYPT_PASSWORD")
	if password == "" {
		return "", errors.New("ENCRYPT_PASSWORD not set")
	}

	// Generate a random 16-byte IV
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	salt := reverse(password)
	// Derive a 32-byte key using scrypt
	key, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 32)
	if err != nil {
		return "", err
	}

	// Create a cipher using AES-256-CTR with the key and IV
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	stream := cipher.NewCTR(block, iv)

	// Encrypt the text
	ciphertext := make([]byte, len(textToEncrypt))
	stream.XORKeyStream(ciphertext, []byte(textToEncrypt))

	// Combine the IV and ciphertext into a single string
	encryptedText := hex.EncodeToString(iv) + hex.EncodeToString(ciphertext)

	return encryptedText, nil
}

func DecryptText(encryptedText string) (string, error) {
	password := os.Getenv("ENCRYPT_PASSWORD")
	if password == "" {
		return "", errors.New("ENCRYPT_PASSWORD not set")
	}

	if len(encryptedText) < 32 {
		return "", errors.New("invalid encrypted text")
	}

	// Extract the IV and ciphertext from the string
	iv, err := hex.DecodeString(encryptedText[:32])
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(encryptedText[32:])
	if err != nil {
		return "", err
	}

	salt := reverse(password)
	// Derive the key using scrypt
	key, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 32)
	if err != nil {
		return "", err
	}

	// Create a decipher using AES-256-CTR with the key and IV
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	stream := cipher.NewCTR(block, iv)

	// Decrypt the ciphertext
	decryptedText := make([]byte, len(ciphertext))
	stream.XORKeyStream(decryptedText, ciphertext)

	return string(decryptedText), nil
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
