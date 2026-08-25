import { describe, expect, test } from '@jest/globals'
import { shouldFailoverRpc } from '../src/settings'

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
