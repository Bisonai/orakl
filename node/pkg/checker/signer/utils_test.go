package signer

import "testing"

func TestWhitelistSignatureGeneratesABI(t *testing.T) {
	name, in, out, err := ParseMethodSignature("whitelist(address) returns ((uint256, uint256))")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	abi, err := GenerateViewABI(name, in, out)
	if err != nil {
		t.Fatalf("abi (in=%q out=%q): %v", in, out, err)
	}
	if len(abi.Methods[name].Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(abi.Methods[name].Outputs))
	}
}
