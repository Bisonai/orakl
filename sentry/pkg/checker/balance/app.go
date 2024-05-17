package balance

import (
	"context"
	"errors"
	"math"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

var SubmitterAlarmAmount float64
var DelegatorAlarmAmount float64
var BalanceCheckInterval time.Duration
var klaytnClient *client.Client
var wallets []Wallet

type LoadedAddress struct {
	Address string `db:"address" json:"address"`
}

type Wallet struct {
	Address    string  `db:"address" json:"address"`
	Balance    float64 `db:"balance" json:"balance"`
	LastChange float64 `db:"last_change" json:"last_change"`
	Minimum    float64
}

func init() {
	var err error
	ctx := context.Background()

	SubmitterAlarmAmount = 25
	DelegatorAlarmAmount = 10000
	BalanceCheckInterval = 10 * time.Second

	submitterAlarmAmountRaw := os.Getenv("SUBMITTER_ALARM_AMOUNT")
	if submitterAlarmAmountRaw != "" && isNumber(submitterAlarmAmountRaw) {
		SubmitterAlarmAmount, _ = strconv.ParseFloat(submitterAlarmAmountRaw, 64)
	}

	delegatorAlarmAmountRaw := os.Getenv("DELEGATOR_ALARM_AMOUNT")
	if delegatorAlarmAmountRaw != "" && isNumber(delegatorAlarmAmountRaw) {
		DelegatorAlarmAmount, _ = strconv.ParseFloat(delegatorAlarmAmountRaw, 64)
	}

	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		log.Error().Msg("JSON_RPC_URL not found")
		panic("JSON_RPC_URL not found")
	}

	klaytnClient, err = client.Dial(jsonRpcUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to klaytn client")
		panic(err)
	}

	interval := os.Getenv("BALANCE_CHECK_INTERVAL")
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse BALANCE_CHECK_INTERVAL, using default 10s")
	} else {
		BalanceCheckInterval = parsedInterval
	}

	oraklDatabaseUrl := os.Getenv("ORAKL_DATABASE_URL")
	if oraklDatabaseUrl == "" {
		log.Error().Msg("ORAKL_DATABASE_URL not found")
		panic("ORAKL_DATABASE_URL not found")
	}

	oraklPool, err := pgxpool.New(ctx, oraklDatabaseUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to orakl database")
		panic(err)
	}
	defer oraklPool.Close()

	nodeDatabaseUrl := os.Getenv("NODE_DATABASE_URL")
	if nodeDatabaseUrl == "" {
		log.Error().Msg("NODE_DATABASE_URL not found")
		panic("NODE_DATABASE_URL not found")
	}

	nodePool, err := pgxpool.New(ctx, nodeDatabaseUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to node database")
		panic(err)
	}
	defer nodePool.Close()

	delegatorAddress := os.Getenv("DELEGATOR_ADDRESS")
	if delegatorAddress == "" {
		panic("DELEGATOR_ADDRESS not found")
	}

	reportersFromOrakl, err := loadAddressesFromOrakl(ctx, oraklPool)
	if err != nil {
		log.Error().Err(err).Msg("Error loading addresses from orakl")
		panic(err)
	}

	reportersFromNode, err := loadAddressesFromNode(ctx, nodePool)
	if err != nil {
		log.Error().Err(err).Msg("Error loading addresses from node")
		panic(err)
	}

	reporters := append(reportersFromOrakl, reportersFromNode...)
	for _, reporter := range reporters {
		balance, err := getBalance(ctx, reporter.Address)
		if err != nil {
			balance = 0
			continue
		}

		wallet := Wallet{
			Address:    reporter.Address,
			Balance:    balance,
			LastChange: 0,
			Minimum:    SubmitterAlarmAmount,
		}
		wallets = append(wallets, wallet)
	}

	delegatorBalance, err := getBalance(ctx, delegatorAddress)
	if err != nil {
		delegatorBalance = 0
	}
	wallets = append(wallets, Wallet{
		Address:    delegatorAddress,
		Balance:    delegatorBalance,
		LastChange: 0,
		Minimum:    DelegatorAlarmAmount,
	})
}

func loadAddressesFromOrakl(ctx context.Context, pool *pgxpool.Pool) ([]LoadedAddress, error) {
	rows, err := pool.Query(ctx, "SELECT address FROM reporters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	addresses, err := pgx.CollectRows(rows, pgx.RowToStructByName[LoadedAddress])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return addresses, err
}

func loadAddressesFromNode(ctx context.Context, pool *pgxpool.Pool) ([]LoadedAddress, error) {
	rows, err := pool.Query(ctx, "SELECT address FROM wallets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	addresses, err := pgx.CollectRows(rows, pgx.RowToStructByName[LoadedAddress])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return addresses, err
}

func getBalance(ctx context.Context, address string) (float64, error) {
	account := common.HexToAddress(address)
	balance, err := klaytnClient.BalanceAt(ctx, account, nil)
	if err != nil {
		return 0, err
	}
	fbalance := new(big.Float).SetInt(balance)
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	result, _ := ethValue.Float64()

	return result, nil
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
