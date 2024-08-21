package tests

import (
	"os"
	"testing"

	"bisonai.com/miko/node/pkg/utils/encryptor"
)

func TestEncryptDecryptText(t *testing.T) {
	// Set the encryption password
	os.Setenv("ENCRYPT_PASSWORD", "testpassword")

	// Define the text to encrypt
	originalText := "Hello, World!"

	// Encrypt the text
	encryptedText, err := encryptor.EncryptText(originalText)
	if err != nil {
		t.Fatalf("Failed to encrypt text: %v", err)
	}

	encryptedText2, err := encryptor.EncryptText(originalText)
	if err != nil {
		t.Fatalf("Failed to encrypt text: %v", err)
	}
	if encryptedText == encryptedText2 {
		t.Fatalf("Encrypted text is the same, shouldn't be idempotent")
	}

	// Decrypt the text
	decryptedText, err := encryptor.DecryptText(encryptedText)
	if err != nil {
		t.Fatalf("Failed to decrypt text: %v", err)
	}

	// Check that the decrypted text matches the original text
	if decryptedText != originalText {
		t.Fatalf("Decrypted text does not match original text. Got %s, want %s", decryptedText, originalText)
	}
}
