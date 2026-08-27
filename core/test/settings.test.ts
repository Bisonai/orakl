import { afterEach, describe, expect, jest, test } from '@jest/globals'
import { ethers } from 'ethers'
import { createJsonRpcProvider, shouldFailoverRpc } from '../src/settings'

describe('shouldFailoverRpc', function () {
  test('retries a transient read failure on the fallback endpoint', function () {
    expect(shouldFailoverRpc('eth_call', { code: 'SERVER_ERROR' })).toBe(true)
    expect(shouldFailoverRpc('eth_getBlockByNumber', { code: 'TIMEOUT' })).toBe(true)
    expect(shouldFailoverRpc('eth_getLogs', { code: 'NETWORK_ERROR' })).toBe(true)
  })

  test('never fails over network detection or transaction submission', function () {
    // eth_chainId must resolve from the primary only (never lock the process to the fallback chain);
    // eth_sendRawTransaction must not be replayed (double-broadcast).
    expect(shouldFailoverRpc('eth_chainId', { code: 'SERVER_ERROR' })).toBe(false)
    expect(shouldFailoverRpc('eth_sendRawTransaction', { code: 'SERVER_ERROR' })).toBe(false)
  })

  test('does not fail over on a real (non-transient) error', function () {
    // A contract revert or a malformed request is a real result, not an outage.
    expect(shouldFailoverRpc('eth_call', { code: 'CALL_EXCEPTION' })).toBe(false)
    expect(shouldFailoverRpc('eth_call', undefined)).toBe(false)
    expect(shouldFailoverRpc('eth_call', {})).toBe(false)
  })
})

describe('FallbackJsonRpcProvider chain guard', function () {
  afterEach(function () {
    jest.restoreAllMocks()
  })

  function stubProvider(primaryChainId: number, fallbackChainId: number) {
    const provider = createJsonRpcProvider('http://primary.invalid', 'http://fallback.invalid')
    // Private field: the secondary provider the guard verifies before use.
    const fallback = (provider as any)._fallbackProvider
    jest
      .spyOn(provider, 'getNetwork')
      .mockResolvedValue({ chainId: primaryChainId, name: 'primary' } as any)
    jest
      .spyOn(fallback, 'getNetwork')
      .mockResolvedValue({ chainId: fallbackChainId, name: 'fallback' } as any)
    const fallbackSend = jest.spyOn(fallback, 'send').mockResolvedValue('0xfallback' as any)
    // Every primary read fails transiently so the guard is exercised.
    jest
      .spyOn(ethers.providers.JsonRpcProvider.prototype, 'send')
      .mockRejectedValue({ code: 'SERVER_ERROR' } as never)
    return { provider, fallbackSend }
  }

  test('rejects instead of using a fallback on a different chain', async function () {
    const { provider, fallbackSend } = stubProvider(1001, 8217)
    await expect(provider.send('eth_blockNumber', [])).rejects.toBeDefined()
    expect(fallbackSend).not.toHaveBeenCalled()
  })

  test('uses the fallback when it reports the same chain', async function () {
    const { provider, fallbackSend } = stubProvider(8217, 8217)
    await expect(provider.send('eth_blockNumber', [])).resolves.toBe('0xfallback')
    expect(fallbackSend).toHaveBeenCalled()
  })
})
