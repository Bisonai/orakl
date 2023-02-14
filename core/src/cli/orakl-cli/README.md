# `orakl-cli`

This package is used controlling [Orakl Network](https://www.orakl.network/) nodes.

## Development

```shell
yarn install
yarn build
```

## How To Use

- [Chain](#chain)
- [Service](#service)
- [Listener](#listener)
- [VRF](#vrf)
- [Adapter](#adapter)
- [Aggregator](#aggregator)
- [Key-Value](#key-value)

### Chain

List all chains

```shell
npx orakl-cli chain list
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
npx orakl-cli chain insert --name other
```

Remove chain specified by `id`

```
npx orakl-cli chain remove --id 4
```

### Service

List all services

```
npx orakl-cli service list
```

```
[
  { id: 1, name: 'VRF' },
  { id: 2, name: 'Aggregator' },
  { id: 3, name: 'RequestResponse' }
]
```

Insert new service

```
npx orakl-cli service insert --name Automation
```

Remove service specified by `id`

```
npx orakl-cli service remove --id 4
```

### Listener

List all listeners

```
npx orakl-cli listener list
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
npx orakl-cli listener list --chain localhost
```

List listeners based on `--service` filter

```
npx orakl-cli listener list --service VRF
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
npx orakl-cli listener insert \
    --chain baobaob \
    --service VRF \
    --address 0x97ba95dcc35e820148cab9ce488f650c77e4736f \
    --eventName SomeEvent
```

Remove listener specified by `id`

```
npx orakl-cli listener remove --id 4
```

## VRF

List all VRF keys

```
npx orakl-cli vrf list
```

List VRF keys based on `--chain` filter

```
npx orakl-cli vrf list --chain localhost
```

Insert new VRF keys for `baobab` chain. VRF keys can be generated with `npx orakl-cli keygen`

```
npx orakl-cli vrf insert \
    --sk 83a8c15d203a71f4f9e7238d663d1ae7eabe10bee47699d4256438acf9bdcce3 \
    --pk 044ffbfebcd28f48144c18f7bd9f233199c438b39b5ce1ecc8f049924ba57a8740a814ca7ac5d14c34850e3b61dcbce296de95a4578ac928f8bab48f2a834d1bb9 \
    --pk_x 36177951785554001241008675842510466823271112960800516449139368880820117473088 \
    --pk_y 76025292965992487548362208012694556435399374398995576443525051210529378212793 \
    --chain baobab
```

Remove VRF keys specified by `id`

```
npx orakl-cli vrf remove --id 2
```

### Adapter

List all adapters

```shell
npx orakl-cli adapter list
```

List all adapters registered for `baobab` chain

```shell
npx orakl-cli adapter list --chain baobab
```

Add new adaper

```shell
npx orakl-cli adapter insert --file-path [adapter-file] --chain baobab
```

Remove adapter

```shell
npx orakl-cli adapter remove --id [id]
```

Add new adapter from other chain

```shell
npx orakl-cli adapter insertFromChain --adapter-id [adapter-id] --from-chain [from-chain] --to-chain [to-chain]
```

### Aggregator

List all aggregators

```shell
npx orakl-cli aggregator list
```

List all aggregators registered for `baobab` chain

```shell
npx orakl-cli aggregator list --chain baobab
```

Add new aggregator

```shell
npx orakl-cli aggregator insert --file-path [aggregator-file] --adapter [adapter-id] --chain baobab
```

Remove aggregator

```shell
npx orakl-cli aggregator remove --id [id]
```

Add new aggregator from other chain

```shell
npx orakl-cli aggregator insertFromChain --aggregator-id [aggregator-id] --adapter [adapter-id] --from-chain [from-chain] --to-chain [to-chain]
```

### Key-Value

List all key-value pairs

```shell
npx orakl-cli kv list
```

List all key-value pairs in `localhost` network

```shell
npx orakl-cli kv list --chain localhost
```

Display value for given key (`PUBLIC_ADDRESS`) in `localhost` network

```shell
npx orakl-cli kv list \
    --key PUBLIC_ADDRESS \
    --chain localhost
```

Insert value (`8888`) for a key (`HEALTH_CHECK_PORT`) ona a `localhost` network

```shell
npx orakl-cli kv insert \
    --key HEALTH_CHECK_PORT \
    --value 8888 \
    --chain localhost
```

If you want to insert key-value pair where value is an empty string, you can omit the `--value` parameter.

```shell
npx orakl-cli kv insert \
    --key SLACK_WEBHOOK_URL \
    --chain localhost
```

Insert many key-value pairs defined in JSON-formatted file

```shell
npx orakl-cli kv insertMany \
    --file-path path/to/file.json \
    --chain localhost
```

Delete key-value pair defined by key (`PUBLIC_ADDRESS`) on a `localhost` network

```shell
npx orakl-cli kv remove \
    --key PUBLIC_ADDRESS \
    --chain localhost
```

Update value (`8888`) for a key (`HEALTH_CHECK_PORT`) ona a `localhost` network

```shell
npx orakl-cli kv udpate \
    --key HEALTH_CHECK_PORT \
    --value 8888 \
    --chain localhost
```

If you want to update key-value pair with a value that is an empty string, you can omit the `--value` parameter.

```shell
npx orakl-cli kv update \
    --key SLACK_WEBHOOK_URL \
    --chain localhost
```
