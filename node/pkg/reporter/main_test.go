//nolint:all
package reporter

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	os.Exit(code)
}
