package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubOnePassword(t *testing.T, value string) *[2]string {
	t.Helper()
	got := new([2]string)
	orig := resolveOnePassword
	resolveOnePassword = func(_ context.Context, token, ref string) (string, error) {
		got[0], got[1] = token, ref
		return value, nil
	}
	t.Cleanup(func() { resolveOnePassword = orig })
	return got
}

func TestLoadFeePayerFromOnePassword(t *testing.T) {
	got := stubOnePassword(t, " 0xabc \n")
	tokenFile := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("ops_file\n"), 0o600))

	t.Setenv(EnvOnePasswordRef, "op://vault/item/field")
	t.Setenv(EnvOnePasswordToken, "ops_env")
	t.Setenv(EnvOnePasswordTokenFile, tokenFile)

	pk, err := LoadFeePayerFromOnePassword(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "abc", pk)
	assert.Equal(t, [2]string{"ops_file", "op://vault/item/field"}, *got, "token file wins over env")

	t.Setenv(EnvOnePasswordTokenFile, "")
	_, err = LoadFeePayerFromOnePassword(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ops_env", got[0])
}

func TestLoadFeePayerFromOnePasswordMissingConfig(t *testing.T) {
	stubOnePassword(t, "abc")
	t.Setenv(EnvOnePasswordRef, "")
	_, err := LoadFeePayerFromOnePassword(context.Background())
	assert.ErrorContains(t, err, EnvOnePasswordRef)

	t.Setenv(EnvOnePasswordRef, "op://vault/item/field")
	t.Setenv(EnvOnePasswordToken, "")
	t.Setenv(EnvOnePasswordTokenFile, "")
	_, err = LoadFeePayerFromOnePassword(context.Background())
	assert.ErrorContains(t, err, "token not set")

	t.Setenv(EnvOnePasswordTokenFile, filepath.Join(t.TempDir(), "missing"))
	_, err = LoadFeePayerFromOnePassword(context.Background())
	assert.ErrorContains(t, err, EnvOnePasswordTokenFile)
}

func TestInitFeePayerPKFromOnePassword(t *testing.T) {
	// a valid secp256k1 key (test-only)
	const pk = "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	stubOnePassword(t, pk)
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "false")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")
	t.Setenv(EnvUseOnePassword, "true")
	t.Setenv(EnvOnePasswordRef, "op://vault/item/field")
	t.Setenv(EnvOnePasswordToken, "ops_env")
	t.Setenv(EnvOnePasswordTokenFile, "")

	prev := CurrentFeePayer()
	t.Cleanup(func() { UpdateFeePayer(prev) })
	require.NoError(t, InitFeePayerPK(context.Background(), nil))
	assert.Equal(t, pk, CurrentFeePayer())
}
