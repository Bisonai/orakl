package utils

import (
	"sync"

	errorsentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia/accounts/abi"
)

type AbiWithFunctionName struct {
	Abi          *abi.ABI
	FunctionName string
}

type AbiCache struct {
	AbiMap map[string]*AbiWithFunctionName
	mu     sync.RWMutex
}

func newAbiCache() *AbiCache {
	return &AbiCache{
		AbiMap: make(map[string]*AbiWithFunctionName),
		mu:     sync.RWMutex{},
	}
}

var AbiCacheInstance = newAbiCache()

func GetAbi(functionSignature string) (*abi.ABI, string, error) {
	res, err := AbiCacheInstance.getAbi(functionSignature)
	if err != nil {
		return nil, "", err
	}
	return res.Abi, res.FunctionName, nil
}

func SetAbi(functionSignature string, abi *abi.ABI, functionName string) {
	AbiCacheInstance.setAbi(functionSignature, &AbiWithFunctionName{Abi: abi, FunctionName: functionName})
}

func (c *AbiCache) getAbi(functionSignature string) (*AbiWithFunctionName, error) {
	c.mu.RLock()
	abi, ok := c.AbiMap[functionSignature]
	c.mu.RUnlock()
	if ok {
		return abi, nil
	}
	return nil, errorsentinel.ErrChainCachedAbiNotFound
}

func (c *AbiCache) setAbi(functionSignature string, abiWithFunctionName *AbiWithFunctionName) {
	c.mu.Lock()
	c.AbiMap[functionSignature] = abiWithFunctionName
	c.mu.Unlock()
}
