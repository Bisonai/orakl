package utils

import (
	"time"
)

const (
	DEFAULT_GAS_LIMIT    = uint64(1200000)
	SELECT_WALLETS_QUERY = "SELECT * FROM wallets;"
)

type Wallet struct {
	ID int64  `db:"id"`
	PK string `db:"pk"`
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
	Id          int64     `json:"id" db:"transaction_id"`
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
