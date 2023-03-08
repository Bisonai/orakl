export const dummyFactory = {
  address: '0x5B7a8096dd24CeDa17F47AE040539dC0566Cd1c9',
  abi: [
    {
      inputs: [],
      stateMutability: 'nonpayable',
      type: 'constructor'
    },
    {
      inputs: [],
      name: 'COUNTER',
      outputs: [
        {
          internalType: 'uint256',
          name: '',
          type: 'uint256'
        }
      ],
      stateMutability: 'view',
      type: 'function'
    },
    {
      inputs: [],
      name: 'decreament',
      outputs: [],
      stateMutability: 'nonpayable',
      type: 'function'
    },
    {
      inputs: [],
      name: 'increament',
      outputs: [],
      stateMutability: 'nonpayable',
      type: 'function'
    }
  ],
  transactionHash: '0x847ecabd654d73e8b3ca1258ed93c985c819bd00232881b1fdc54352d483c3b1',
  receipt: {
    to: null,
    from: '0x30E30C3B6313FF232E93593b883fC8A8AF8BB627',
    contractAddress: '0x5B7a8096dd24CeDa17F47AE040539dC0566Cd1c9',
    transactionIndex: 0,
    gasUsed: '180435',
    logsBloom:
      '0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
    blockHash: '0x1386e6edea1f6b8b82b0b8c7104692651168d6d89db4e099385a47974e3f1272',
    transactionHash: '0x847ecabd654d73e8b3ca1258ed93c985c819bd00232881b1fdc54352d483c3b1',
    logs: [],
    blockNumber: 116593393,
    cumulativeGasUsed: '180435',
    status: 1,
    byzantium: true
  },
  args: [],
  numDeployments: 1,
  solcInputHash: '18e4179754a3342d95612f8d7efd3fb0',
  metadata:
    '{"compiler":{"version":"0.8.16+commit.07a7930e"},"language":"Solidity","output":{"abi":[{"inputs":[],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[],"name":"COUNTER","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"decreament","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"increament","outputs":[],"stateMutability":"nonpayable","type":"function"}],"devdoc":{"kind":"dev","methods":{},"version":1},"userdoc":{"kind":"user","methods":{},"version":1}},"settings":{"compilationTarget":{"src/v0.1/Dummy.sol":"Dummy"},"evmVersion":"london","libraries":{},"metadata":{"bytecodeHash":"ipfs","useLiteralContent":true},"optimizer":{"enabled":true,"runs":1000},"remappings":[]},"sources":{"src/v0.1/Dummy.sol":{"content":"// SPDX-License-Identifier: MIT\\npragma solidity ^0.8.16;\\n\\ncontract Dummy{\\n    \\n    uint256 public COUNTER = 0;\\n    constructor() {}\\n\\n    function increament() public {\\n        COUNTER = COUNTER + 1;\\n    }\\n\\n    function decreament() public {\\n        require(COUNTER > 0, \\"Minimum\\");\\n        COUNTER = COUNTER - 1;\\n    }\\n}\\n","keccak256":"0x396743a5df97941d2c8c8eb761eced08be3651212572d601139aefdca39e6d9d","license":"MIT"}},"version":1}',
  bytecode:
    '0x60806040526000805534801561001457600080fd5b50610195806100246000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806342541b8b1461004657806358968e881461005057806399b7057914610058575b600080fd5b61004e610073565b005b61004e6100f6565b61006160005481565b60405190815260200160405180910390f35b60008054116100e2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600760248201527f4d696e696d756d00000000000000000000000000000000000000000000000000604482015260640160405180910390fd5b60016000546100f19190610133565b600055565b6000546100f190600161014c565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561014657610146610104565b92915050565b808201808211156101465761014661010456fea26469706673582212207960e5423f5ad859ea26055344e71c1f29c321ecd8b76f066dbbcfc0101b655e64736f6c63430008100033',
  deployedBytecode:
    '0x608060405234801561001057600080fd5b50600436106100415760003560e01c806342541b8b1461004657806358968e881461005057806399b7057914610058575b600080fd5b61004e610073565b005b61004e6100f6565b61006160005481565b60405190815260200160405180910390f35b60008054116100e2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600760248201527f4d696e696d756d00000000000000000000000000000000000000000000000000604482015260640160405180910390fd5b60016000546100f19190610133565b600055565b6000546100f190600161014c565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561014657610146610104565b92915050565b808201808211156101465761014661010456fea26469706673582212207960e5423f5ad859ea26055344e71c1f29c321ecd8b76f066dbbcfc0101b655e64736f6c63430008100033',
  devdoc: {
    kind: 'dev',
    methods: {},
    version: 1
  },
  userdoc: {
    kind: 'user',
    methods: {},
    version: 1
  },
  storageLayout: {
    storage: [
      {
        astId: 4,
        contract: 'src/v0.1/Dummy.sol:Dummy',
        label: 'COUNTER',
        offset: 0,
        slot: '0',
        type: 't_uint256'
      }
    ],
    types: {
      t_uint256: {
        encoding: 'inplace',
        label: 'uint256',
        numberOfBytes: '32'
      }
    }
  }
}
