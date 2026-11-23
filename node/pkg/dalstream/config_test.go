package dalstream

import (
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv(EnvDALURL, "http://dal:8090/")
	t.Setenv(EnvDALAPIKey, "k")
	t.Setenv(EnvTick, "2s")
	t.Setenv(EnvMaxConnsPerIP, "3")
	t.Setenv(EnvAllowedOrigins, "https://orakl.network")
	t.Setenv(EnvTrustedProxies, "10.0.0.1")
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DALURL != "http://dal:8090" || cfg.Tick != 2*time.Second || cfg.MaxConnsPerIP != 3 || cfg.MaxConns != 1000 ||
		len(cfg.AllowedOrigins) != 1 || len(cfg.TrustedProxies) != 1 || cfg.Addr != ":8080" || cfg.MetricsAddr != ":8081" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if got := wsURL(cfg.DALURL); got != "ws://dal:8090/ws" {
		t.Fatalf("wsURL = %q", got)
	}
	if got := wsURL("https://dal.example"); got != "wss://dal.example/ws" {
		t.Fatalf("wsURL = %q", got)
	}
}

func TestConfigFromEnvErrors(t *testing.T) {
	t.Setenv(EnvDALURL, "")
	t.Setenv(EnvDALAPIKey, "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("missing DAL_URL/DAL_API_KEY accepted")
	}
	t.Setenv(EnvDALURL, "dal:8090")
	t.Setenv(EnvDALAPIKey, "k")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("URL without scheme accepted")
	}
	t.Setenv(EnvDALURL, "http://dal:8090")
	for name, v := range map[string]string{EnvTick: "0s", EnvMaxConns: "-1", EnvTrustedProxies: "x"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, v)
			if _, err := ConfigFromEnv(); err == nil {
				t.Fatalf("%s=%q accepted", name, v)
			}
		})
	}
}
