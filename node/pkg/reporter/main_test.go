//nolint:all
package reporter

import (
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}
