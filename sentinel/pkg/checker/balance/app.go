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

	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
)

const (
	oraklApiEndpoint       = "/reporter"
	oraklNodeAdminEndpoint = "/wallet/addresses"
	oraklDelegatorEndpoint = "/sign/feePayer"
	porEndpoint            = "/address"
)

var SubmitterAlarmAmount float64
var DelegatorAlarmAmount float64
var BalanceCheckInterval time.Duration
var BalanceAlarmInterval time.Duration

var klaytnClient *client.Client
var wallets []Wallet

type Urls struct {
	JsonRpcUrl        string
	OraklApiUrl       string
	OraklNodeAdminUrl string
	OraklDelegatorUrl string
	PorUrl            string
}

type Wallet struct {
	Address common.Address `db:"address" json:"address"`
	Balance float64        `db:"balance" json:"balance"`
	Minimum float64
}

func init() {
	loadEnvs()
}

func setUp() {
	ctx := context.Background()

	urls, err := getUrls()
	if err != nil {
		log.Error().Err(err).Msg("Error getting urls")
		panic(err)
	}

	err = setClient(urls.JsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error setting up client")
		panic(err)
	}

	wallets, err = loadWallets(ctx, urls)
	if err != nil {
		log.Error().Err(err).Msg("Error loading wallets")
		panic(err)
	}
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

func Start(ctx context.Context) {
	setUp()
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
	BalanceCheckInterval = 10 * time.Second
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
	wallets := []Wallet{}

	apiWallets, err := loadWalletFromOraklApi(ctx, urls.OraklApiUrl)
	if err != nil {
		return wallets, err
	}
	wallets = append(wallets, apiWallets...)

	adminWallets, err := loadWalletFromOraklAdmin(ctx, urls.OraklNodeAdminUrl)
	if err != nil {
		return wallets, err
	}
	wallets = append(wallets, adminWallets...)

	porWallet, err := loadWalletFromPor(ctx, urls.PorUrl)
	if err != nil {
		return wallets, err
	}
	wallets = append(wallets, porWallet)

	delegatorWallet, err := loadWalletFromDelegator(ctx, urls.OraklDelegatorUrl)
	if err != nil {
		return wallets, err
	}
	wallets = append(wallets, delegatorWallet)

	return wallets, nil
}

func loadWalletFromOraklApi(ctx context.Context, url string) ([]Wallet, error) {
	type ReporterModel struct {
		Address string `json:"address"`
	}

	wallets := []Wallet{}
	reporters, err := request.GetRequest[[]ReporterModel](url+oraklApiEndpoint, nil, nil)
	if err != nil {
		return wallets, err
	}

	for _, reporter := range reporters {
		address := common.HexToAddress(reporter.Address)

		wallet := Wallet{
			Address: address,
			Balance: 0,
			Minimum: SubmitterAlarmAmount,
		}
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

func loadWalletFromOraklAdmin(ctx context.Context, url string) ([]Wallet, error) {
	wallets := []Wallet{}
	reporters, err := request.GetRequest[[]string](url+oraklNodeAdminEndpoint, nil, nil)
	if err != nil {
		return wallets, err
	}

	for _, reporter := range reporters {
		address := common.HexToAddress(reporter)
		wallet := Wallet{
			Address: address,
			Balance: 0,
			Minimum: SubmitterAlarmAmount,
		}
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

func loadWalletFromPor(ctx context.Context, url string) (Wallet, error) {
	wallet := Wallet{}
	reporter, err := request.GetRequest[string](url+porEndpoint, nil, nil)
	if err != nil {
		return wallet, err
	}
	address := common.HexToAddress(reporter)
	wallet = Wallet{
		Address: address,
		Balance: 0,
		Minimum: SubmitterAlarmAmount,
	}
	return wallet, nil
}

func loadWalletFromDelegator(ctx context.Context, url string) (Wallet, error) {
	wallet := Wallet{}
	feePayer, err := request.GetRequest[string](url+oraklDelegatorEndpoint, nil, nil)
	if err != nil {
		return wallet, err
	}
	address := common.HexToAddress(feePayer)
	wallet = Wallet{
		Address: address,
		Balance: 0,
		Minimum: DelegatorAlarmAmount,
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
	for i, wallet := range wallets {
		time.Sleep(1 * time.Second) //gracefully request to prevent json rpc blockage
		balance, err := getBalance(ctx, wallet.Address)
		if err != nil {
			log.Error().Err(err).Str("address", wallet.Address.Hex()).Msg("Error getting balance")
			continue
		}
		wallets[i].Balance = balance
	}
}

func alarm(wallets []Wallet) {
	var alarmMessage = ""
	for _, wallet := range wallets {
		if wallet.Balance < wallet.Minimum {
			log.Error().Str("address", wallet.Address.Hex()).Float64("balance", wallet.Balance).Msg("Balance lower than minimum")
			alarmMessage += fmt.Sprintf("%s balance(%f) is lower than minimum(%f)\n", wallet.Address.Hex(), wallet.Balance, wallet.Minimum)
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
