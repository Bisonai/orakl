package secrets

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

type SecretEnv struct {
	VaultRole       string
	JwtPath         string
	VaultSecretPath string
	VaultKeyName    string
}

type Secrets struct {
	FeePayer string
}

func (s *SecretEnv) GetSecretFromVaultWithKubernetesAuth() (*Secrets, error) {
	ctx := context.Background()
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		s.VaultRole,
		auth.WithServiceAccountTokenPath(s.JwtPath),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Kubernetes auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(ctx, k8sAuth)
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

	// comma-ok: an unchecked assertion here panicked when the secret came back
	// without FEE_PAYER. this runs on the fee payer retry goroutine, where a
	// panic is unrecoverable and would kill the process.
	raw, ok := secrets.Data["FEE_PAYER"]
	if !ok {
		return nil, fmt.Errorf("FEE_PAYER missing from vault secret %s/%s", s.VaultSecretPath, s.VaultKeyName)
	}
	feePayer, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("FEE_PAYER in vault secret %s/%s is %T, want string", s.VaultSecretPath, s.VaultKeyName, raw)
	}

	return &Secrets{FeePayer: feePayer}, nil
}
