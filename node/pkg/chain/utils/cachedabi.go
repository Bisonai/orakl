package utils

import (
	"sync"

	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn/accounts/abi"
)

type AbiWithFunctionName struct {
	Abi          *abi.ABI
	FunctionName string
}

type AbiCache struct {
	AbiMap map[string]*AbiWithFunctionName
	mu     sync.RWMutex
}

func NewAbiCache() *AbiCache {
	return &AbiCache{
		AbiMap: make(map[string]*AbiWithFunctionName),
		mu:     sync.RWMutex{},
	}
}

var AbiCacheInstance = NewAbiCache()

func GetAbi(functionSignature string) (*abi.ABI, string, error) {
	res, err := AbiCacheInstance.GetAbi(functionSignature)
	if err != nil {
		return nil, "", err
	}
	return res.Abi, res.FunctionName, nil
}

func SetAbi(functionSignature string, abi *abi.ABI, functionName string) {
	AbiCacheInstance.SetAbi(functionSignature, &AbiWithFunctionName{Abi: abi, FunctionName: functionName})
}

func (c *AbiCache) GetAbi(functionSignature string) (*AbiWithFunctionName, error) {
	c.mu.RLock()
	abi, ok := c.AbiMap[functionSignature]
	c.mu.RUnlock()
	if ok {
		return abi, nil
	}
	return nil, errorsentinel.ErrChainCachedAbiNotFound
}

func (c *AbiCache) SetAbi(functionSignature string, abiWithFunctionName *AbiWithFunctionName) {
	c.mu.Lock()
	c.AbiMap[functionSignature] = abiWithFunctionName
	c.mu.Unlock()
}
