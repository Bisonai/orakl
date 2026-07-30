package utils

import (
	"strings"
	"testing"
)

// validKey is a throwaway secp256k1 key used only to exercise validation.
const validKey = "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestValidateFeePayerPK(t *testing.T) {
	tests := []struct {
		name    string
		pk      string
		wantErr bool
	}{
		{name: "valid key", pk: validKey, wantErr: false},
		{name: "valid key with 0x prefix", pk: "0x" + validKey, wantErr: false},
		// the exact shape that took both networks down: the vault lookup
		// returned nothing and the empty string was accepted as a key
		{name: "empty", pk: "", wantErr: true},
		{name: "truncated", pk: validKey[:32], wantErr: true},
		{name: "too long", pk: validKey + "ab", wantErr: true},
		{name: "not hex", pk: strings.Repeat("z", 64), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeePayerPK(tt.pk)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateFeePayerPK(%q) = nil, want error", tt.pk)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateFeePayerPK(%q) = %v, want nil", tt.pk, err)
			}
		})
	}
}

// setFeePayer must leave the stored key untouched when handed a bad one, so a
// failed reload cannot destroy a working key.
func TestSetFeePayerRejectsBadKeyAndKeepsPrevious(t *testing.T) {
	t.Cleanup(func() { UpdateFeePayer("") })

	if err := setFeePayer(validKey); err != nil {
		t.Fatalf("setFeePayer(valid) = %v, want nil", err)
	}
	if got := CurrentFeePayer(); got != validKey {
		t.Fatalf("CurrentFeePayer() = %q, want %q", got, validKey)
	}

	if err := setFeePayer(""); err == nil {
		t.Fatal("setFeePayer(\"\") = nil, want error")
	}
	if got := CurrentFeePayer(); got != validKey {
		t.Fatalf("CurrentFeePayer() = %q after failed reload, want previous key retained", got)
	}
}

func TestSetFeePayerStripsPrefix(t *testing.T) {
	t.Cleanup(func() { UpdateFeePayer("") })

	if err := setFeePayer("0x" + validKey); err != nil {
		t.Fatalf("setFeePayer(prefixed) = %v, want nil", err)
	}
	if got := CurrentFeePayer(); got != validKey {
		t.Fatalf("CurrentFeePayer() = %q, want prefix stripped", got)
	}
}
