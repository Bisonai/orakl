package loginit

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const defaultLogLevel = zerolog.InfoLevel

// resolveLevel maps a LOG_LEVEL value to a zerolog level. Empty or unset
// falls back to the default; zerolog.ParseLevel("") returns NoLevel without
// an error, which would otherwise silence every line. invalid reports a
// non-empty value that could not be parsed.
func resolveLevel(raw string) (level zerolog.Level, invalid bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLogLevel, false
	}
	level, err := zerolog.ParseLevel(raw)
	if err != nil || level == zerolog.NoLevel {
		return defaultLogLevel, true
	}
	return level, false
}

func InitZeroLog() {
	raw := os.Getenv("LOG_LEVEL")
	level, invalid := resolveLevel(raw)
	zerolog.SetGlobalLevel(level)
	if invalid {
		log.Warn().Str("LOG_LEVEL", raw).Str("fallback", level.String()).Msg("invalid LOG_LEVEL, using default")
	}
	// NoLevel event: printed at every level except disabled
	log.Log().Str("logLevel", level.String()).Msg("log level set")
}
