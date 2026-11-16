package balance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"

	"bisonai.com/miko/node/pkg/alert"
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

	alarmTicker := time.NewTicker(BalanceAlarmInterval)
	defer alarmTicker.Stop()

	for {
		select {
		case <-checkTicker.C:
			updateBalances(ctx, wallets)
		case <-alarmTicker.C:
			alarm(wallets)
		}
	}
}

func loadEnvs() {
	BalanceCheckInterval = 60 * time.Second
	BalanceAlarmInterval = 30 * time.Minute

	checkInterval := os.Getenv("BALANCE_CHECK_INTERVAL")
	parsedCheckInterval, err := time.ParseDuration(checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse BALANCE_CHECK_INTERVAL, using default 10s")
	} else {
		BalanceCheckInterval = parsedCheckInterval
	}

	alarmInterval := os.Getenv("BALANCE_ALARM_INTERVAL")
	parsedAlarmInterval, err := time.ParseDuration(alarmInterval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse BALANCE_ALARM_INTERVAL, using default 10s")
	} else {
		BalanceAlarmInterval = parsedAlarmInterval
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
	cutoff := time.Now().Add(-BalanceHistoryTTL)
	for i, wallet := range wallets {
		time.Sleep(500 * time.Millisecond) //gracefully request to prevent json rpc blockage
		balance, err := getBalance(ctx, wallet.Address)
		if err != nil {
			rpcErrorsCounter.Inc()
			log.Error().Err(err).Str("address", wallet.Address.Hex()).Msg("Error getting balance")
			continue
		}
		recordBalance(wallet, balance, time.Now())
		log.Debug().Str("address", wallet.Address.Hex()).Float64("balance", balance).Str("tag", wallet.Tag).Msg("Updated balance")
		wallets[i].Balance = balance

		balanceHistoryEntry := BalanceHistoryEntry{
			Timestamp: time.Now(),
			Balance:   balance,
		}
		wallets[i].BalanceHistory = append(wallets[i].BalanceHistory, balanceHistoryEntry)

		var recentHistory []BalanceHistoryEntry
		for _, entry := range wallets[i].BalanceHistory {
			if entry.IsRecent(cutoff) {
				recentHistory = append(recentHistory, entry)
			}
		}
		wallets[i].BalanceHistory = recentHistory

		log.Debug().
			Str("address", wallet.Address.Hex()).
			Float64("balance", balance).
			Str("tag", wallet.Tag).
			Msg("Updated balance and recorded history")
	}
}

func alarm(wallets []Wallet) {
	var alarmMessage = ""
	for _, wallet := range wallets {
		log.Debug().Str("address", wallet.Address.Hex()).Float64("balance", wallet.Balance).Float64("minimum", wallet.Minimum).Str("tag", wallet.Tag).Msg(wallet.Tag)
		if wallet.Balance < wallet.Minimum {
			log.Warn().Str("address", wallet.Address.Hex()).Float64("balance", wallet.Balance).Msg("Balance lower than minimum")
			alarmMessage += fmt.Sprintf("%s balance(%f) is lower than minimum(%f) | %s\n", wallet.Address.Hex(), wallet.Balance, wallet.Minimum, wallet.Tag)
		}

		if len(wallet.BalanceHistory) < 30 {
			continue
		}

		latestDrainage := wallet.BalanceHistory[len(wallet.BalanceHistory)-1].Balance - wallet.BalanceHistory[len(wallet.BalanceHistory)-ChunkSize].Balance
		averageDrainage := getAverageDrainage(wallet.BalanceHistory)

		if latestDrainage > averageDrainage || averageDrainage == 0 {
			continue
		}

		drainageChangeRatio := (averageDrainage - latestDrainage) / math.Abs(averageDrainage)
		if drainageChangeRatio > MinimalIncreaseThreshold {
			log.Warn().Str("address", wallet.Address.Hex()).Float64("latestDrainage", latestDrainage).Float64("averageDrainage", averageDrainage).Msg("Increased balance")
			alarmMessage += fmt.Sprintf("%s balance drained faster by %.2f%% | %s\n", wallet.Address.Hex(), drainageChangeRatio*100, wallet.Tag)
		}
	}

	if alarmMessage != "" {
		alert.SlackAlert(alarmMessage)
	}
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func getAverageDrainage(history []BalanceHistoryEntry) float64 {
	drainageList := []float64{}
	for i := 0; i < len(history); i += ChunkSize {
		if i+ChunkSize-1 >= len(history) {
			break
		}
		drainage := history[i+ChunkSize-1].Balance - history[i].Balance
		if drainage < 0 { // Only consider negative drainage
			drainageList = append(drainageList, drainage)
		}
	}

	if len(drainageList) == 0 {
		return 0
	}

	sum := float64(0)
	for _, value := range drainageList {
		sum += value
	}
	return sum / float64(len(drainageList))
}
