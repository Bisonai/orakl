package utils

import (
	"context"
	"math/big"
	"time"

	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
)

const (
	DEFAULT_MINE_WAIT_TIME     = 10 * time.Second
	DEFAULT_GAS_LIMIT          = uint64(6000000)
	SELECT_PROVIDER_URLS_QUERY = "SELECT * FROM provider_urls WHERE chain_id = @chain_id ORDER BY priority;"
	LOAD_SIGNER                = "SELECT id, pk FROM signer LIMIT 1;"
	STORE_SIGNER               = "INSERT INTO signer (pk, unique_dummy) VALUES (@pk, TRUE) ON CONFLICT (unique_dummy) DO UPDATE SET pk = EXCLUDED.pk;"

	// signer_key keyring (issue #2516). The keyring durably holds every private key the
	// node may still need (the confirmed active key + any in-flight pending + retired
	// history) so a key that is / becomes the on-chain oracle is never lost across a crash.
	LOAD_SIGNER_KEYS   = "SELECT id, address, pk, state, tx_hash, created_at, updated_at FROM signer_key ORDER BY created_at;"
	INSERT_PENDING_KEY = "INSERT INTO signer_key (address, pk, state) VALUES (@address, @pk, 'pending') RETURNING id;"
	BACKFILL_ADDRESS   = "UPDATE signer_key SET address = @address, updated_at = now() WHERE id = @id AND address IS NULL;"
	SET_KEY_TXHASH     = "UPDATE signer_key SET tx_hash = @tx_hash, updated_at = now() WHERE id = @id;"
	// PromotePendingToActive runs RETIRE_ACTIVE then ACTIVATE_KEY in one transaction, so the
	// signer_key_one_active partial unique index is never transiently violated.
	RETIRE_ACTIVE = "UPDATE signer_key SET state = 'retired', updated_at = now() WHERE state = 'active' AND id <> @id;"
	ACTIVATE_KEY  = "UPDATE signer_key SET state = 'active', updated_at = now() WHERE id = @id;"
	GC_RETIRED    = "DELETE FROM signer_key WHERE state = 'retired' AND updated_at < now() - interval '90 days';"
)

type Wallet struct {
	ID int64  `db:"id"`
	PK string `db:"pk"`
}

type SignerKey struct {
	ID        int64      `db:"id"`
	Address   *string    `db:"address"`
	PK        string     `db:"pk"` // encrypted at rest
	State     string     `db:"state"`
	TxHash    *string    `db:"tx_hash"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type ProviderUrl struct {
	ID       *int64 `db:"id"`
	ChainId  *int   `db:"chain_id"`
	Url      string `db:"url"`
	Priority *int   `db:"priority"`
}

type SignInsertPayload struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Input    string `json:"input"`
	Gas      string `json:"gas"`
	Value    string `json:"value"`
	ChainId  string `json:"chainId"`
	GasPrice string `json:"gasPrice"`
	Nonce    string `json:"nonce"`
	V        string `json:"v"`
	R        string `json:"r"`
	S        string `json:"s"`
	RawTx    string `json:"rawTx"`
}

type SignModel struct {
	ID          int64     `json:"id" db:"transaction_id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	From        string    `json:"from" db:"from"`
	To          string    `json:"to" db:"to"`
	Input       string    `json:"input" db:"input"`
	Gas         string    `json:"gas" db:"gas"`
	Value       string    `json:"value" db:"value"`
	ChainId     string    `json:"chainId" db:"chainId"`
	GasPrice    string    `json:"gasPrice" db:"gasPrice"`
	Nonce       string    `json:"nonce" db:"nonce"`
	V           string    `json:"v" db:"v"`
	R           string    `json:"r" db:"r"`
	S           string    `json:"s" db:"s"`
	RawTx       string    `json:"rawTx" db:"rawTx"`
	SignedRawTx *string   `json:"signedRawTx" db:"signedRawTx"`
	Succeed     *bool     `json:"succeed" db:"succeed"`
	FunctionId  int64     `json:"functionId" db:"functionId"`
	ContractId  int64     `json:"contractId" db:"contractId"`
	ReporterId  int64     `json:"reporterId" db:"reporterId"`
}

type ClientInterface interface {
	Close()
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, call kaia.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	CallContract(ctx context.Context, call kaia.CallMsg, blockNumber *big.Int) ([]byte, error)
	NetworkID(ctx context.Context) (*big.Int, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	BlockNumber(ctx context.Context) (*big.Int, error)
	SubscribeFilterLogs(ctx context.Context, q kaia.FilterQuery, ch chan<- types.Log) (kaia.Subscription, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (kaia.Subscription, error)
}

type JsonRpcError interface {
	Error() string
	ErrorCode() int
	ErrorData() interface{}
}
