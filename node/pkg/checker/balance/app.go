package balance

import (
	"context"
	"errors"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func init() {
	loadEnvs()
}

func setUp() error {
	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		return errors.New("JSON_RPC_URL not found")
	}

	err := setClient(jsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error setting up client")
		return err
	}

	wallets, err = ParseWallets(os.Getenv(EnvWallets))
	if err != nil {
		log.Error().Err(err).Msg("Error loading wallets")
		return err
	}
	registerWalletMetrics(wallets)
	log.Info().Int("wallets", len(wallets)).Msg("Loaded balance wallets")
	return nil
}

func setClient(jsonRpcUrl string) error {
	var err error
	kaiaClient, err = client.Dial(jsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to kaia client")
		return err
	}
	return nil
}

func Start(ctx context.Context) error {
	err := setUp()
	if err != nil {
		return err
	}
	log.Info().Msg("Starting balance checker")
	checkTicker := time.NewTicker(BalanceCheckInterval)
	defer checkTicker.Stop()

	// Alerts: Prometheus rules on the exported orakl_balance_* metrics.
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-checkTicker.C:
			updateBalances(ctx, wallets)
		}
	}
}

func loadEnvs() {
	BalanceCheckInterval = 60 * time.Second

	checkInterval := os.Getenv("BALANCE_CHECK_INTERVAL")
	parsedCheckInterval, err := time.ParseDuration(checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse BALANCE_CHECK_INTERVAL, using default 10s")
	} else {
		BalanceCheckInterval = parsedCheckInterval
	}

}

func getBalance(ctx context.Context, address common.Address) (float64, error) {
	balance, err := kaiaClient.BalanceAt(ctx, address, nil)
	if err != nil {
		return 0, err
	}
	fbalance := new(big.Float).SetInt(balance)
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	result, _ := ethValue.Float64()

	return result, nil
}

func updateBalances(ctx context.Context, wallets []Wallet) {
	for i, wallet := range wallets {
		time.Sleep(500 * time.Millisecond) //gracefully request to prevent json rpc blockage
		balance, err := getBalance(ctx, wallet.Address)
		if err != nil {
			rpcErrorsCounter.Inc()
			log.Error().Err(err).Str("address", wallet.Address.Hex()).Msg("Error getting balance")
			continue
		}
		recordBalance(wallet, balance, time.Now())
		wallets[i].Balance = balance
		log.Debug().Str("address", wallet.Address.Hex()).Float64("balance", balance).Str("role", wallet.Tag).Msg("Updated balance")
	}
}
