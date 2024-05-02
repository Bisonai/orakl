package secrets

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

type SecretEnv struct {
	VaultRole       string
	VaultTokenPath  string
	VaultSecretPath string
	VaultKeyName    string
}

type Secrets struct {
	DatabaseURL     string
	EncryptPassword string
}

func (s *SecretEnv) GetSecretFromVaultWithKubernetesAuth() (*Secrets, error) {
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		s.VaultRole,
		auth.WithServiceAccountTokenPath(s.VaultTokenPath),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Kubernetes auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to log in with Kubernetes auth: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}

	secrets, err := client.KVv2(s.VaultSecretPath).Get(context.Background(), s.VaultKeyName)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret: %w", err)
	}

	secretDataSet := &Secrets{
		DatabaseURL:     secrets.Data["DATABASE_URL"].(string),
		EncryptPassword: secrets.Data["ENCRYPT_PASSWORD"].(string),
	}

	return secretDataSet, nil
}
