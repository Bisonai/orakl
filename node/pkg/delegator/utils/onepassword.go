package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	onepassword "github.com/1password/onepassword-sdk-go"
)

const (
	// EnvUseOnePassword selects 1Password as the fee payer source.
	EnvUseOnePassword = "USE_1PASSWORD"
	// EnvOnePasswordRef is the secret reference of the fee payer key,
	// e.g. op://<vault>/<item>/<field>.
	EnvOnePasswordRef = "OP_FEEPAYER_REF"
	// EnvOnePasswordToken / EnvOnePasswordTokenFile give the service account
	// token directly or as a file path (preferred: keeps it out of the env).
	EnvOnePasswordToken     = "OP_SERVICE_ACCOUNT_TOKEN"
	EnvOnePasswordTokenFile = "OP_SERVICE_ACCOUNT_TOKEN_FILE"
)

// resolveOnePassword is indirected so tests can avoid the 1Password API.
var resolveOnePassword = func(ctx context.Context, token, ref string) (string, error) {
	client, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("orakl-delegator", "v1"))
	if err != nil {
		return "", err
	}
	return client.Secrets().Resolve(ctx, ref)
}

func onePasswordToken() (string, error) {
	if path := os.Getenv(EnvOnePasswordTokenFile); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", EnvOnePasswordTokenFile, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return os.Getenv(EnvOnePasswordToken), nil
}

// LoadFeePayerFromOnePassword reads the fee payer key from 1Password with a
// service account. The key is held in memory only.
func LoadFeePayerFromOnePassword(ctx context.Context) (string, error) {
	ref := os.Getenv(EnvOnePasswordRef)
	if ref == "" {
		return "", errors.New(EnvOnePasswordRef + " not set")
	}
	token, err := onePasswordToken()
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", errors.New("1Password service account token not set (" + EnvOnePasswordTokenFile + " or " + EnvOnePasswordToken + ")")
	}
	pk, err := resolveOnePassword(ctx, token, ref)
	if err != nil {
		return "", fmt.Errorf("resolve fee payer from 1Password: %w", err)
	}
	return strings.TrimPrefix(strings.TrimSpace(pk), "0x"), nil
}
