export const ADAPTER_0 = {
  active: true,
  name: 'X-Y',
  decimals: 8,
  adapterHash: '0x020e150749af3bffaec9ae337da0b9b00c3cfe0b46b854a8e2f5922f6ba2c5db',
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
          { function: 'ROUND' }
        ]
      }
    }
  ]
}

export const ADAPTER_1 = {
  active: true,
  name: 'Z-X',
  decimals: 8,
  adapterHash: '0x12da2f5119ba624ed025303b424d637349c0d120d02bd66a9cfff57e98463a81',
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
          { function: 'ROUND' }
        ]
      }
    }
  ]
}

export const AGGREGATOR_0 = {
  name: 'X-Y',
  aggregatorHash: '0x5bcc6c18d584dc54a666f9212229226f02f65b8dcda3ed72836b6c901f2d18e1',
  address: '0x0000000000000000000000000000000000000000',
  heartbeat: 15000,
  threshold: 0.05,
  absoluteThreshold: 0.1,
  adapterHash: '0x020e150749af3bffaec9ae337da0b9b00c3cfe0b46b854a8e2f5922f6ba2c5db'
}

export const AGGREGATOR_1 = {
  name: 'Z-X',
  aggregatorHash: '0x11ca65b539221125a64b38653f65dbbf961ed2ea16bcaf54408a5d2ebdc13a0b',
  address: '0x0000000000000000000000000000000000000001',
  heartbeat: 15000,
  threshold: 0.05,
  absoluteThreshold: 0.1,
  adapterHash: '0x12da2f5119ba624ed025303b424d637349c0d120d02bd66a9cfff57e98463a81'
}

export const VRF_0 = {
  chain: 'baobab',
  sk: 'adcfaf9a860722a89472884a2aab4a62f06a42fd4bee55f2fc7f2f11b07f1d81',
  pk: '041f058731839e8c2fb3a77a4be788520f1743f1298a84bd138871f31ffdee04e42b4f962995ba0135eed67f3ebd1739d4b09f1b84224c0d6765e5f426b25443a4',
  pkX: '14031465612060486287063884409830887522455901523026705297854775800516553082084',
  pkY: '19590069790275828365845547074408283587257770205538752975574862882950389973924',
  keyHash: '0x956506aeada5568c80c984b908e9e1af01bd96709977b0b5cb1957736e80e883'
}

export const VRF_1 = {
  chain: 'localhost',
  sk: '123',
  pk: '456',
  pkX: '789',
  pkY: '101112',
  keyHash: '0x'
}

export const DATAFEED_BULK_0 = {
  bulk: [
    {
      adapterSource: 'https://config.orakl.network/adapter/ada-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/ada-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xa',
        walletPrivateKey: '0xb'
      }
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/atom-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/atom-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xc',
        walletPrivateKey: '0xd'
      }
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/avax-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/avax-usdt.aggregator.json',
      reporter: {
        walletAddress: '0xe',
        walletPrivateKey: '0xf'
      }
    }
  ]
}

export const DATAFEED_BULK_1 = {
  chain: 'baobab',
  service: 'DATA_FEED_V2',
  organization: 'kf',
  functionName: 'submitV2(uint256, int256)',
  eventName: 'NewRoundV2',
  bulk: [
    {
      adapterSource: 'https://config.orakl.network/adapter/bnb-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/bnb-usdt.aggregator.json',
      reporter: {
        walletAddress: '0x0',
        walletPrivateKey: '0x1'
      }
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/bora-krw.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/bora-krw.aggregator.json',
      reporter: {
        walletAddress: '0x2',
        walletPrivateKey: '0x3'
      }
    },
    {
      adapterSource: 'https://config.orakl.network/adapter/eth-usdt.adapter.json',
      aggregatorSource: 'https://config.orakl.network/aggregator/baobab/eth-usdt.aggregator.json',
      reporter: {
        walletAddress: '0x4',
        walletPrivateKey: '0x5'
      }
    }
  ]
}
