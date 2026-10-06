package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"cypress": "mainnet",
		"baobab":  "kairos",
		"mainnet": "mainnet",
		"kairos":  "kairos",
		"":        "",
		"unknown": "unknown",
	}
	for in, want := range cases {
		assert.Equal(t, want, Normalize(in), "Normalize(%q)", in)
	}
}
