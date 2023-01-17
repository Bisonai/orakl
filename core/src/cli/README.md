## CLI

### Prerequisities

```
yarn build
```

### Core Operator

#### Adapter

* Add new adapter
* Activate adapter
* Deactivate adapter

#### Aggregator

* Add new aggregator
* Activate aggregator
* Deactivate aggregator

#### Chain

List all chains

```
yarn cli chain list
```

```
[
  { id: 1, name: 'localhost' },
  { id: 2, name: 'baobab' },
  { id: 3, name: 'cypress' }
]
```

Insert new chain

```
yarn cli chain insert --name other
```

Remove chain by `id` filter

```
yarn cli chain remove --id 4
```

#### Listener

List all listeners

```
yarn cli listener list
```

```
[
  {
    address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
    eventName: 'RandomWordsRequested'
  },
  {
    address: '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853',
    eventName: 'NewRound'
  },
  {
    address: '0x45778c29A34bA00427620b937733490363839d8C',
    eventName: 'Requested'
  }
]
```

List listeners based on `--chain` filter

```
yarn cli listener list --chain localhost
```

List listeners based on `--service` filter

```
yarn cli listener list --service VRF
```

```
[
  {
    address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
    eventName: 'RandomWordsRequested'
  }
]
```

Insert new listener to `baobab` chain

```
yarn cli listener insert \
    --chain baobaob \
    --service VRF \
    --address 0x97ba95dcc35e820148cab9ce488f650c77e4736f \
    --eventName SomeEvent
```

Remove listener based on `id` filter

```
yarn cli listener remove --id 4
```


### VRF

List all VRF keys

```
yarn cli vrf list
```

List VRF keys based on `--chain` filter

```
yarn cli vrf list --chain localhost
```

Insert new VRF keys for `baobab` chain. VRF keys can be generated with `yarn keygen`

```
yarn cli vrf insert \
    --sk 83a8c15d203a71f4f9e7238d663d1ae7eabe10bee47699d4256438acf9bdcce3 \
    --pk 044ffbfebcd28f48144c18f7bd9f233199c438b39b5ce1ecc8f049924ba57a8740a814ca7ac5d14c34850e3b61dcbce296de95a4578ac928f8bab48f2a834d1bb9 \
    --pk_x 36177951785554001241008675842510466823271112960800516449139368880820117473088 \
    --pk_y 76025292965992487548362208012694556435399374398995576443525051210529378212793 \
    --chain baobab
```

Remove VRF keys based on `id` filter.

```
yarn cli vrf remove --id 2
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
