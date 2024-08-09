package secrets

import (
	"context"
	"os"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/rs/zerolog/log"
)

var secretData map[string]interface{}
var initialized bool = false

func init() {
	ctx := context.Background()

	vaultRole := os.Getenv("VAULT_ROLE")
	jwtPath := os.Getenv("JWT_PATH")
	vaultSecretPath := os.Getenv("VAULT_SECRET_PATH")
	vaultKeyName := os.Getenv("VAULT_KEY_NAME")

	if vaultRole == "" || jwtPath == "" || vaultSecretPath == "" || vaultKeyName == "" {
		log.Error().Msg("Missing required environment variables for Vault initialization")
		return
	}

	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		log.Error().Err(err).Msg("unable to initialize Vault client")
		return
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultRole,
		auth.WithServiceAccountTokenPath(jwtPath),
	)
	if err != nil {
		log.Error().Err(err).Msg("unable to initialize Kubernetes auth method")
		return
	}

	authInfo, err := client.Auth().Login(ctx, k8sAuth)
	if err != nil {
		log.Error().Err(err).Msg("unable to log in with Kubernetes auth")
		return
	}
	if authInfo == nil {
		log.Error().Err(err).Msg("no auth info was returned after login")
		return
	}

	secrets, err := client.KVv2(vaultSecretPath).Get(ctx, vaultKeyName)
	if err != nil {
		log.Error().Err(err).Msg("unable to read secret")
		return
	}

	secretData = secrets.Data
	initialized = true
}

func GetSecret(key string) string {
	if !initialized {
		return os.Getenv(key)
	}
	value, ok := secretData[key]
	if !ok {
		return os.Getenv(key)
	}
	result, ok := value.(string)
	if !ok || result == "" {
		return os.Getenv(key)
	}
	return result
}
