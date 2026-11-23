package loginit

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Documents the zerolog behavior this package guards against.
func TestZerologParseEmptyIsNoLevel(t *testing.T) {
	level, err := zerolog.ParseLevel("")
	if err != nil || level != zerolog.NoLevel {
		t.Fatalf("expected NoLevel without error, got %v, %v", level, err)
	}
}

func TestResolveLevel(t *testing.T) {
	cases := []struct {
		raw     string
		want    zerolog.Level
		invalid bool
	}{
		{"", zerolog.InfoLevel, false},
		{"   ", zerolog.InfoLevel, false},
		{"debug", zerolog.DebugLevel, false},
		{"info", zerolog.InfoLevel, false},
		{"warn", zerolog.WarnLevel, false},
		{" error ", zerolog.ErrorLevel, false},
		{"trace", zerolog.TraceLevel, false},
		{"disabled", zerolog.Disabled, false},
		{"verbose", zerolog.InfoLevel, true},
		{"1", zerolog.InfoLevel, false},
		{"10", zerolog.InfoLevel, true},
		{"-5", zerolog.InfoLevel, true},
	}
	for _, tc := range cases {
		got, invalid := resolveLevel(tc.raw)
		if got != tc.want || invalid != tc.invalid {
			t.Errorf("resolveLevel(%q) = %v, %v; want %v, %v", tc.raw, got, invalid, tc.want, tc.invalid)
		}
	}
}

func captureInit(t *testing.T, value string, set bool) string {
	t.Helper()
	prevLogger, prevLevel := log.Logger, zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = prevLogger
		zerolog.SetGlobalLevel(prevLevel)
	})
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf)
	if set {
		t.Setenv("LOG_LEVEL", value)
	} else {
		t.Setenv("LOG_LEVEL", "")
		os.Unsetenv("LOG_LEVEL")
	}
	InitZeroLog()
	return buf.String()
}

func TestInitZeroLogEmptyAndUnsetDefaultToInfo(t *testing.T) {
	for _, set := range []bool{true, false} {
		out := captureInit(t, "", set)
		if zerolog.GlobalLevel() != zerolog.InfoLevel {
			t.Fatalf("set=%v: global level = %v, want info", set, zerolog.GlobalLevel())
		}
		if !strings.Contains(out, `"logLevel":"info"`) {
			t.Fatalf("set=%v: missing startup line, got %q", set, out)
		}
		if strings.Contains(out, "invalid LOG_LEVEL") {
			t.Fatalf("set=%v: unexpected invalid warning: %q", set, out)
		}
	}
}

func TestInitZeroLogInvalidWarns(t *testing.T) {
	out := captureInit(t, "verbose", true)
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Fatalf("global level = %v, want info", zerolog.GlobalLevel())
	}
	if !strings.Contains(out, "invalid LOG_LEVEL") || !strings.Contains(out, `"LOG_LEVEL":"verbose"`) {
		t.Fatalf("missing invalid warning naming value, got %q", out)
	}
}

func TestInitZeroLogStartupLineAtWarn(t *testing.T) {
	out := captureInit(t, "warn", true)
	if zerolog.GlobalLevel() != zerolog.WarnLevel {
		t.Fatalf("global level = %v, want warn", zerolog.GlobalLevel())
	}
	if !strings.Contains(out, `"logLevel":"warn"`) {
		t.Fatalf("startup line must print above info too, got %q", out)
	}
}
