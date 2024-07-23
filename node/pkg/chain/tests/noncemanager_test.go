//nolint:all
package tests

import (
	"sync"
	"testing"

	"bisonai.com/orakl/node/pkg/chain/noncemanager"
)

func TestNonceManager(t *testing.T) {
	nm := noncemanager.New()
	address := "0x123"

	nm.SetNonce(address, 1)
	nonce, err := nm.GetNonce(address)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if nonce != 1 {
		t.Fatalf("Expected nonce to be 1, got %d", nonce)
	}

	newNonce, err := nm.GetNonceAndIncrement(address)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if newNonce != 1 {
		t.Fatalf("Expected new nonce to be 2, got %d", newNonce)
	}

	nonce, err = nm.GetNonce(address)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if nonce != 2 {
		t.Fatalf("Expected nonce to be 2, got %d", nonce)
	}

	newAddress := "0x456"
	_, err = nm.GetNonceAndIncrement(newAddress)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	wg := sync.WaitGroup{}
	concurrentIncrements := 100
	for i := 0; i < concurrentIncrements; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.GetNonceAndIncrement(address)
		}()
	}
	wg.Wait()

	nonce, err = nm.GetNonce(address)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expectedNonce := uint64(concurrentIncrements + 2)
	if nonce != expectedNonce {
		t.Fatalf("Expected nonce to be %d, got %d", expectedNonce, nonce)
	}
}
