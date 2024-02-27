package tests

import (
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/utils"
	"github.com/go-playground/assert/v2"
)

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv("ENCRYPT_PASSWORD", "mysecretpassword")

	testText := "myTestTextItIs"
	encryptedText, err := utils.EncryptText(testText)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, encryptedText, testText)

	decryptedText, err := utils.DecryptText(encryptedText)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, decryptedText, encryptedText)
	assert.Equal(t, decryptedText, testText)
}
