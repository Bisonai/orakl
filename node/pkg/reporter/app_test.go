// //nolint:all
package reporter

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/testutil/testenv"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
)

func TestFetchConfigs(t *testing.T) {
	configs, err := fetchConfigs()
	if err != nil || configs == nil || len(configs) == 0 {
		t.Fatalf("error getting configs: %v", err)
	}
}

func TestRunApp(t *testing.T) {
	testenv.Env(t, dalAppEnv...)
	ctx := context.Background()

	app := New()
	err := app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}
}

// TestWsDataHandling checks that a signed submission delivered by the DAL
// websocket ends up, intact, in the reporter's LatestDataMap. It runs against
// an in-process DAL (see newMockDal) and a throwaway signer key, so it needs
// no chain, secrets or live DAL.
func TestWsDataHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("error generating signer key: %v", err)
	}
	signerPK := hex.EncodeToString(crypto.FromECDSA(signerKey))
	whitelist := []kaiacommon.Address{crypto.PubkeyToAddress(signerKey.PublicKey)}

	config := types.Config{ID: 13, Name: "test-aggregate"}
	dal, err := newMockDal(t, ctx, config, whitelist)
	if err != nil {
		t.Fatalf("error mocking dal ws server: %v", err)
	}

	app := New()
	// The production helper: it subscribes to every config on (re)connect.
	app.WsHelper, err = SetupDalWsHelper(ctx, []Config{{Name: config.Name}}, dal.wsURL, dal.apiKey)
	if err != nil {
		t.Fatalf("error setting up dal ws helper: %v", err)
	}
	go app.WsHelper.Run(ctx, app.HandleWsMessage)

	timestamp := time.Now()
	sample, err := generateSampleSubmissionData(signerPK, config.ID, int64(15), timestamp, 1, config.Name)
	if err != nil {
		t.Fatalf("error generating sample submission data: %v", err)
	}
	// What the DAL collector delivers for a verified submission.
	outgoing, err := dal.collector.IncomingDataToOutgoingData(ctx, sample)
	if err != nil {
		t.Fatalf("error converting sample submission data: %v", err)
	}
	stream := dal.collector.OutgoingStream[config.Name]

	// The hub delivers only to clients already subscribed, and the reporter
	// subscribes asynchronously (about a second after dialing). Resend until
	// the data arrives instead of guessing how long that takes; the cap only
	// bounds a real failure.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second)
	for {
		if got, ok := GetLatestData(app.LatestDataMap, config.Name); ok {
			if got.Symbol != config.Name || got.Value != 15 || got.AggregateTime != timestamp.UnixMilli() {
				t.Fatalf("unexpected submission data: %+v", got)
			}
			if !bytes.Equal(got.Proof, sample.Proof.Proof) {
				t.Fatalf("proof mismatch: got %x, want %x", got.Proof, sample.Proof.Proof)
			}
			if !bytes.Equal(got.FeedHash[:], crypto.Keccak256([]byte(config.Name))) {
				t.Fatalf("feed hash mismatch: got %x", got.FeedHash)
			}
			return
		}
		select {
		case <-ticker.C:
			select {
			case stream <- outgoing:
			default:
			}
		case <-timeout:
			t.Fatal("submission data not received from websocket")
		}
	}
}
