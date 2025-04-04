package loginit

import (
	"os"

	"github.com/rs/zerolog"
)

const defaultLogLevel = zerolog.WarnLevel

func InitZeroLog() {
	logLvStr := os.Getenv("LOG_LEVEL")
	level, err := zerolog.ParseLevel(logLvStr)
	if err != nil {
		level = defaultLogLevel
	}
	zerolog.SetGlobalLevel(level)
}
