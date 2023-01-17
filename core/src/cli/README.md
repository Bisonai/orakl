## CLI

### Prerequisities

```
yarn build
```

### Generate Adapter Hash

```
yarn adapter-hash [--verify] [file ...]
```

Verify all adapters from `ADAPTER_ROOT_DIR`

```
yarn adapter-hash --verify
```

Generate adapter with updated hash for KLAY/USD

```
yarn adapter-hash adapter/klay_usd.adapter.json
```

```
{
  id: '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5',
  active: true,
  name: 'KLAY/USD',
  jobType: 'DATA_FEED',
  decimals: '8',
  feeds: [
    {
      url: 'https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD',
      headers: { 'Content-Type': 'application/json' },
      method: 'GET',
      reducers: [
        { function: 'PARSE', args: [ 'RAW', 'KLAY', 'USD', 'PRICE' ] },
        { function: 'POW10', args: '8' },
        { function: 'ROUND' }
      ]
    },
    {
      url: 'https://api.coingecko.com/api/v3/simple/price?ids=klay-token&vs_currencies=usd',
      headers: { 'Content-Type': 'application/json' },
      method: 'GET',
      reducers: [
        { function: 'PARSE', args: [ 'klay-token', 'usd' ] },
        { function: 'POW10', args: '8' },
        { function: 'ROUND' }
      ]
    },
    {
      url: 'https://api.coinbase.com/v2/exchange-rates?currency=KLAY',
      headers: { 'Content-Type': 'application/json' },
      method: 'GET',
      reducers: [
        { function: 'PARSE', args: [ 'data', 'rates', 'USD' ] },
        { function: 'POW10', args: '8' },
        { function: 'ROUND' }
      ]
    }
  ]
}
```

### Generate Aggregator Hash

```
yarn aggregator-hash [--verify] [file ...]
```

Verify all aggregators from `AGGREGATOR_ROOT_DIR`

```
yarn adapter-hash --verify
```

Generate aggregator with updated hash for KLAY/USD

```
yarn aggregator-hash aggregator/klay_usd.aggregator.json
```

```
{
  id: '0x0f283adfbb1eb73ad0098689bd0595b56265c77d24f620866f945fb2d849b123',
  active: true,
  name: 'KLAY/USD',
  fixedHeartbeatRate: { active: true, value: 5000 },
  randomHeartbeatRate: { active: false, value: 2000 },
  threshold: 0.05,
  absoluteThreshold: 0.1,
  adapterId: '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5'
}
```
