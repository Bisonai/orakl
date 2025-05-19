package eth_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/api"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/common/hexutil"
	"github.com/kaiachain/kaia/networks/rpc"
	"github.com/kaiachain/kaia/rlp"
)

type EthClient struct {
	c       *rpc.Client
	chainID *big.Int
}

func Dial(rawurl string) (*EthClient, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*EthClient, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

func NewClient(c *rpc.Client) *EthClient {
	return &EthClient{c, nil}
}

func (ec *EthClient) Close() {
	ec.c.Close()
}

func (ec *EthClient) SetHeader(key, value string) {
	ec.c.SetHeader(key, value)
}

func (ec *EthClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return ec.getBlock(ctx, "eth_getBlockByHash", hash, true)
}

func (ec *EthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return ec.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), true)
}

type rpcBlock struct {
	Hash         common.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
}

func (ec *EthClient) getBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {
	var raw json.RawMessage
	err := ec.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, kaia.NotFound
	}

	var head *types.Header
	var body rpcBlock
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs), nil
}

func (ec *EthClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	var head *types.Header
	err := ec.c.CallContext(ctx, &head, "eth_getBlockByHash", hash, false)
	if err == nil && head == nil {
		err = kaia.NotFound
	}
	return head, err
}

func (ec *EthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := ec.c.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = kaia.NotFound
	}
	return head, err
}

func (ec *EthClient) SubscribeFilterLogs(ctx context.Context, q kaia.FilterQuery, ch chan<- types.Log) (kaia.Subscription, error) {
	return ec.c.Subscribe(ctx, "eth", ch, "logs", toFilterArg(q))
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

func (ec *EthClient) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	var json *rpcTransaction
	err = ec.c.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, kaia.NotFound
	} else if sigs := json.tx.RawSignatureValues(); sigs[0].V == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, json.BlockNumber == nil, nil
}

func (ec *EthClient) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	if tx == nil {
		return common.Address{}, errors.New("Transaction must not be nil")
	}
	sender, err := types.Sender(&senderFromServer{blockhash: block}, tx)
	if err == nil {
		return sender, nil
	}
	var meta struct {
		Hash common.Hash
		From common.Address
	}
	if err = ec.c.CallContext(ctx, &meta, "eth_getTransactionByBlockHashAndIndex", block, hexutil.Uint64(index)); err != nil {
		return common.Address{}, err
	}
	if meta.Hash == (common.Hash{}) || meta.Hash != tx.Hash() {
		return common.Address{}, errors.New("wrong inclusion block/index")
	}
	return meta.From, nil
}

func (ec *EthClient) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	var num hexutil.Uint
	err := ec.c.CallContext(ctx, &num, "eth_getBlockTransactionCountByHash", blockHash)
	return uint(num), err
}

func (ec *EthClient) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	var json *rpcTransaction
	err := ec.c.CallContext(ctx, &json, "eth_getTransactionByBlockHashAndIndex", blockHash, hexutil.Uint64(index))
	if err != nil {
		return nil, err
	}
	if json == nil {
		return nil, kaia.NotFound
	} else if sigs := json.tx.RawSignatureValues(); sigs[0].V == nil {
		return nil, fmt.Errorf("server returned transaction without signature")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, err
}

func (ec *EthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := ec.c.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil {
		if r == nil {
			return nil, kaia.NotFound
		}
	}
	return r, err
}

func (ec *EthClient) TransactionReceiptRpcOutput(ctx context.Context, txHash common.Hash) (r map[string]interface{}, err error) {
	err = ec.c.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil && r == nil {
		return nil, kaia.NotFound
	}
	return
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
}

func (ec *EthClient) SyncProgress(ctx context.Context) (*kaia.SyncProgress, error) {
	var raw json.RawMessage
	if err := ec.c.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}

	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil
	}
	var progress *rpcProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}
	return &kaia.SyncProgress{
		StartingBlock: uint64(progress.StartingBlock),
		CurrentBlock:  uint64(progress.CurrentBlock),
		HighestBlock:  uint64(progress.HighestBlock),
		PulledStates:  uint64(progress.PulledStates),
		KnownStates:   uint64(progress.KnownStates),
	}, nil
}

func (ec *EthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (kaia.Subscription, error) {
	return ec.c.Subscribe(ctx, "eth", ch, "newHeads")
}

func (ec *EthClient) NetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := ec.c.CallContext(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

func (ec *EthClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := ec.c.CallContext(ctx, &result, "eth_getBalance", account, toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

func (ec *EthClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "eth_getStorageAt", account, key, toBlockNumArg(blockNumber))
	return result, err
}

func (ec *EthClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "eth_getCode", account, toBlockNumArg(blockNumber))
	return result, err
}

func (ec *EthClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "eth_getTransactionCount", account, toBlockNumArg(blockNumber))
	return uint64(result), err
}

func (ec *EthClient) FilterLogs(ctx context.Context, q kaia.FilterQuery) ([]types.Log, error) {
	var result []types.Log
	err := ec.c.CallContext(ctx, &result, "eth_getLogs", toFilterArg(q))
	return result, err
}

func toFilterArg(q kaia.FilterQuery) interface{} {
	arg := map[string]interface{}{
		"fromBlock": toBlockNumArg(q.FromBlock),
		"toBlock":   toBlockNumArg(q.ToBlock),
		"address":   q.Addresses,
		"topics":    q.Topics,
	}
	if q.FromBlock == nil {
		arg["fromBlock"] = "0x0"
	}
	return arg
}

func (ec *EthClient) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	var result hexutil.Big
	err := ec.c.CallContext(ctx, &result, "eth_getBalance", account, "pending")
	return (*big.Int)(&result), err
}

func (ec *EthClient) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "eth_getStorageAt", account, key, "pending")
	return result, err
}

func (ec *EthClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "eth_getCode", account, "pending")
	return result, err
}

func (ec *EthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (ec *EthClient) PendingTransactionCount(ctx context.Context) (uint, error) {
	var num hexutil.Uint
	err := ec.c.CallContext(ctx, &num, "eth_getBlockTransactionCountByNumber", "pending")
	return uint(num), err
}

func (ec *EthClient) CallContract(ctx context.Context, msg kaia.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(ctx, &hex, "eth_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (ec *EthClient) PendingCallContract(ctx context.Context, msg kaia.CallMsg) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(ctx, &hex, "eth_call", toCallArg(msg), "pending")
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (ec *EthClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := ec.c.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

func (ec *EthClient) EstimateGas(ctx context.Context, msg kaia.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := ec.c.CallContext(ctx, &hex, "eth_estimateGas", toCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

func (ec *EthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	_, err := ec.SendRawTransaction(ctx, tx)
	return err
}

func (ec *EthClient) SendRawTransaction(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	var hex hexutil.Bytes
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return common.Hash{}, err
	}
	if err := ec.c.CallContext(ctx, &hex, "eth_sendRawTransaction", hexutil.Encode(data)); err != nil {
		return common.Hash{}, err
	}
	hash := common.BytesToHash(hex)
	return hash, nil
}

func (ec *EthClient) SendUnsignedTransaction(ctx context.Context, from common.Address, to common.Address, gas uint64, gasPrice uint64, value *big.Int, data []byte, input []byte) (common.Hash, error) {
	var hex hexutil.Bytes

	tGas := hexutil.Uint64(gas)
	bigGasPrice := new(big.Int).SetUint64(gasPrice)
	tGasPrice := (*hexutil.Big)(bigGasPrice)
	hValue := (*hexutil.Big)(value)
	tData := hexutil.Bytes(data)
	tInput := hexutil.Bytes(input)

	unsignedTx := api.SendTxArgs{
		From:      from,
		Recipient: &to,
		GasLimit:  &tGas,
		Price:     tGasPrice,
		Amount:    hValue,
		Data:      &tData,
		Payload:   &tInput,
	}

	if err := ec.c.CallContext(ctx, &hex, "eth_sendTransaction", toSendTxArgs(unsignedTx)); err != nil {
		return common.Hash{}, err
	}
	hash := common.BytesToHash(hex)
	return hash, nil
}

func (ec *EthClient) ImportRawKey(ctx context.Context, key string, password string) (common.Address, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "personal_importRawKey", key, password)
	address := common.BytesToAddress(result)
	return address, err
}

func (ec *EthClient) UnlockAccount(ctx context.Context, address common.Address, password string, time uint) (bool, error) {
	var result bool
	err := ec.c.CallContext(ctx, &result, "personal_unlockAccount", address, password, time)
	return result, err
}

func toCallArg(msg kaia.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}

func toSendTxArgs(msg api.SendTxArgs) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.Recipient,
	}
	if *msg.GasLimit != 0 {
		arg["gas"] = (*hexutil.Uint64)(msg.GasLimit)
	}
	if msg.Price != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.Price)
	}
	if msg.Amount != nil {
		arg["value"] = (*hexutil.Big)(msg.Amount)
	}
	if len(*msg.Data) > 0 {
		arg["data"] = (*hexutil.Bytes)(msg.Data)
	}
	if len(*msg.Payload) > 0 {
		arg["input"] = (*hexutil.Bytes)(msg.Payload)
	}

	return arg
}

func (ec *EthClient) BlockNumber(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := ec.c.CallContext(ctx, &result, "eth_blockNumber")
	return (*big.Int)(&result), err
}

func (ec *EthClient) ChainID(ctx context.Context) (*big.Int, error) {
	if ec.chainID != nil {
		return ec.chainID, nil
	}

	var result hexutil.Big
	err := ec.c.CallContext(ctx, &result, "eth_chainId")
	if err == nil {
		ec.chainID = (*big.Int)(&result)
	}
	return ec.chainID, err
}

func (ec *EthClient) AddPeer(ctx context.Context, url string) (bool, error) {
	var result bool
	err := ec.c.CallContext(ctx, &result, "admin_addPeer", url)
	return result, err
}

func (ec *EthClient) RemovePeer(ctx context.Context, url string) (bool, error) {
	var result bool
	err := ec.c.CallContext(ctx, &result, "admin_removePeer", url)
	return result, err
}

func (ec *EthClient) CreateAccessList(ctx context.Context, msg kaia.CallMsg) (*types.AccessList, uint64, string, error) {
	type AccessListResult struct {
		Accesslist *types.AccessList `json:"accessList"`
		Error      string            `json:"error,omitempty"`
		GasUsed    hexutil.Uint64    `json:"gasUsed"`
	}
	var result AccessListResult
	if err := ec.c.CallContext(ctx, &result, "eth_createAccessList", toCallArg(msg)); err != nil {
		return nil, 0, "", err
	}
	return result.Accesslist, uint64(result.GasUsed), result.Error, nil
}
