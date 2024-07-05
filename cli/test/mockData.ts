export const ADAPTER_0 = {
  active: true,
  name: 'X-Y',
  decimals: 8,
  adapterHash: '0x78514506aa9d275a66ad8c2480ca60769ba2c597dcd28742e80c50bae56f59ca',
  feeds: [
    {
      name: 'data-X-Y',
      definition: {
        url: 'https://data.com',
        headers: { 'Content-Type': 'application/json' },
        method: 'GET',
        reducers: [
          { function: 'PARSE', args: ['PRICE'] },
          { function: 'POW10', args: '8' },
          { function: 'ROUND' },
        ],
      },
    },
  ],
}

export const ADAPTER_1 = {
  active: true,
  name: 'Z-X',
  decimals: 8,
  adapterHash: '0xfdcd2236964e2d7b7e308ff3f0631e4a0e12df3c1f6eae896279c5c10d4a90c7',
  feeds: [
    {
      name: 'data-Z-X',
      definition: {
        url: 'https://data.com',
        headers: { 'Content-Type': 'application/json' },
        method: 'GET',
        reducers: [
          { function: 'PARSE', args: ['PRICE'] },
          { function: 'POW10', args: '8' },
          { function: 'ROUND' },
        ],
      },
    },
  ],
}

export const AGGREGATOR_0 = {
  name: 'X-Y',
  aggregatorHash: '0xf49b12c34c575369168e0ca822653186546343f7abcbd6ae6fc6c8325bec1f52',
  address: '0x0000000000000000000000000000000000000000',
  heartbeat: 15000,
  threshold: 0.05,
  absoluteThreshold: 0.1,
  adapterHash: '0x78514506aa9d275a66ad8c2480ca60769ba2c597dcd28742e80c50bae56f59ca',
}

export const AGGREGATOR_1 = {
  name: 'Z-X',
  aggregatorHash: '0x3cd7a87af54adcced76d090d975212f0974fd531b1c982b10db4ca22323da30b',
  address: '0x0000000000000000000000000000000000000001',
  heartbeat: 15000,
  threshold: 0.05,
  absoluteThreshold: 0.1,
  adapterHash: '0xfdcd2236964e2d7b7e308ff3f0631e4a0e12df3c1f6eae896279c5c10d4a90c7',
}

export const VRF_0 = {
  chain: 'baobab',
  sk: 'adcfaf9a860722a89472884a2aab4a62f06a42fd4bee55f2fc7f2f11b07f1d81',
  pk: '041f058731839e8c2fb3a77a4be788520f1743f1298a84bd138871f31ffdee04e42b4f962995ba0135eed67f3ebd1739d4b09f1b84224c0d6765e5f426b25443a4',
  pkX: '14031465612060486287063884409830887522455901523026705297854775800516553082084',
  pkY: '19590069790275828365845547074408283587257770205538752975574862882950389973924',
  keyHash: '0x956506aeada5568c80c984b908e9e1af01bd96709977b0b5cb1957736e80e883',
}

export const VRF_1 = {
  chain: 'localhost',
  sk: '123',
  pk: '456',
  pkX: '789',
  pkY: '101112',
  keyHash: '0x',
}

export const DATAFEED_BULK_0 = {
  bulk: [
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/ada-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/ada-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xa',
        walletPrivateKey: '0xb',
      },
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/atom-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/atom-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xc',
        walletPrivateKey: '0xd',
      },
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/avax-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/avax-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xe',
        walletPrivateKey: '0xf',
      },
    },
  ],
}

export const DATAFEED_BULK_1 = {
  chain: 'baobab',
  service: 'DATA_FEED_V2',
  organization: 'kf',
  functionName: 'submitV2(uint256,int256)',
  eventName: 'NewRoundV2',
  bulk: [
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/btc-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/btc-usdt.aggregator.json',
      reporter: {
        walletAddress: '0x0',
        walletPrivateKey: '0x1',
      },
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/ltc-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/ltc-usdt.aggregator.json',
      reporter: {
        walletAddress: '0x2',
        walletPrivateKey: '0x3',
      },
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/baobab/klay-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/klay-usdt.aggregator.json',
      reporter: {
        walletAddress: '0x4',
        walletPrivateKey: '0x5',
      },
    },
  ],
}
