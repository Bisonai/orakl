package balance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/utils/request"
)

func init() {
	loadEnvs()
}

func setUp() error {
	ctx := context.Background()

	urls, err := getUrls()
	if err != nil {
		log.Error().Err(err).Msg("Error getting urls")
		return err
	}

	err = setClient(urls.JsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error setting up client")
		return err
	}

	wallets, err = loadWallets(ctx, urls)
	if err != nil {
		log.Error().Err(err).Msg("Error loading wallets")
		return err
	}
	return nil
}

func setClient(jsonRpcUrl string) error {
	var err error
	klaytnClient, err = client.Dial(jsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to klaytn client")
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
	SubmitterAlarmAmount = 25
	DelegatorAlarmAmount = 10000
	if os.Getenv("CHAIN") == "cypress" {
		DelegatorAlarmAmount = 50000
	}
	BalanceCheckInterval = 60 * time.Second
	BalanceAlarmInterval = 30 * time.Minute

	submitterAlarmAmountRaw := os.Getenv("SUBMITTER_ALARM_AMOUNT")
	if submitterAlarmAmountRaw != "" && isNumber(submitterAlarmAmountRaw) {
		SubmitterAlarmAmount, _ = strconv.ParseFloat(submitterAlarmAmountRaw, 64)
	}

	delegatorAlarmAmountRaw := os.Getenv("DELEGATOR_ALARM_AMOUNT")
	if delegatorAlarmAmountRaw != "" && isNumber(delegatorAlarmAmountRaw) {
		DelegatorAlarmAmount, _ = strconv.ParseFloat(delegatorAlarmAmountRaw, 64)
	}

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

func getUrls() (Urls, error) {
	oraklApiUrl := os.Getenv("ORAKL_API_URL")
	if oraklApiUrl == "" {
		return Urls{}, errors.New("ORAKL_API_URL not found")
	}

	oraklNodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if oraklNodeAdminUrl == "" {
		return Urls{}, errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	oraklDelegatorUrl := os.Getenv("ORAKL_DELEGATOR_URL")
	if oraklDelegatorUrl == "" {
		return Urls{}, errors.New("ORAKL_DELEGATOR_URL not found")
	}

	porUrl := os.Getenv("POR_URL")
	if porUrl == "" {
		return Urls{}, errors.New("POR_URL not found")
	}

	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		return Urls{}, errors.New("JSON_RPC_URL not found")
	}

	return Urls{
		JsonRpcUrl:        jsonRpcUrl,
		OraklApiUrl:       oraklApiUrl,
		OraklNodeAdminUrl: oraklNodeAdminUrl,
		OraklDelegatorUrl: oraklDelegatorUrl,
		PorUrl:            porUrl,
	}, nil
}

func loadWallets(ctx context.Context, urls Urls) ([]Wallet, error) {
	result := []Wallet{}

	apiWallets, err := loadWalletFromOraklApi(ctx, urls.OraklApiUrl)
	if err != nil {
		return result, err
	}
	result = append(result, apiWallets...)

	porWallet, err := loadWalletFromPor(ctx, urls.PorUrl)
	if err != nil {
		return result, err
	}
	result = append(result, porWallet)

	delegatorWallet, err := loadWalletFromDelegator(ctx, urls.OraklDelegatorUrl)
	if err != nil {
		return result, err
	}
	result = append(result, delegatorWallet)

	return result, nil
}

func loadWalletFromOraklApi(ctx context.Context, url string) ([]Wallet, error) {
	type ReporterModel struct {
		Address string `json:"address"`
		Service string `json:"service"`
	}

	result := []Wallet{}
	reporters, err := request.Request[[]ReporterModel](request.WithEndpoint(url+oraklApiEndpoint), request.WithTimeout(30*time.Second))
	if err != nil {
		return result, err
	}

	for _, reporter := range reporters {
		if reporter.Service == "DATA_FEED" {
			continue
		}

		address := common.HexToAddress(reporter.Address)
		minimumBalance := SubmitterAlarmAmount
		if reporter.Service == "REQUEST_RESPONSE" {
			minimumBalance = DefaultRRMinimum
		}

		wallet := Wallet{
			Address: address,
			Balance: 0,
			Minimum: minimumBalance,
			Tag:     "reporter loaded from orakl api",

			BalanceHistory: []BalanceHistoryEntry{},
		}
		result = append(result, wallet)
	}
	return result, nil
}

func loadWalletFromPor(ctx context.Context, url string) (Wallet, error) {
	wallet := Wallet{}
	resp, err := request.RequestRaw(request.WithEndpoint(url + porEndpoint))
	if err != nil {
		return wallet, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Info().
			Int("status", resp.StatusCode).
			Str("url", url+porEndpoint).
			Str("func", "loadWalletFromPor").
			Msg("failed to make request")
		return wallet, errors.New("status not okay from por wallet request")
	}

	reporter, err := io.ReadAll(resp.Body)
	if err != nil {
		return wallet, err
	}

	address := common.HexToAddress(string(reporter))
	wallet = Wallet{
		Address: address,
		Balance: 0,
		Minimum: SubmitterAlarmAmount,
		Tag:     "reporter loaded from por",

		BalanceHistory: []BalanceHistoryEntry{},
	}
	return wallet, nil
}

func loadWalletFromDelegator(ctx context.Context, url string) (Wallet, error) {
	wallet := Wallet{}
	feePayer, err := request.Request[string](request.WithEndpoint(url + oraklDelegatorEndpoint))
	if err != nil {
		return wallet, err
	}
	address := common.HexToAddress(feePayer)
	wallet = Wallet{
		Address: address,
		Balance: 0,
		Minimum: DelegatorAlarmAmount,
		Tag:     "reporter loaded from delegator",

		BalanceHistory: []BalanceHistoryEntry{},
	}
	return wallet, nil
}

func getBalance(ctx context.Context, address common.Address) (float64, error) {
	balance, err := klaytnClient.BalanceAt(ctx, address, nil)
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
			log.Error().Err(err).Str("address", wallet.Address.Hex()).Msg("Error getting balance")
			continue
		}
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

		if len(wallet.BalanceHistory) < 2 {
			continue
		}

		latestDrainage := wallet.BalanceHistory[len(wallet.BalanceHistory)-1].Balance - wallet.BalanceHistory[len(wallet.BalanceHistory)-2].Balance
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
	if len(history) < 2 {
		return 0
	}
	drainageList := []float64{}
	for i := 1; i < len(history)-1; i++ {
		drainage := history[i].Balance - history[i-1].Balance
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
