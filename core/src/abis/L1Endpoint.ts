export const L1EndpointAbis = [
  {
    inputs: [
      {
        internalType: 'address',
        name: 'coordinator',
        type: 'address'
      },
      {
        internalType: 'address',
        name: 'registryAddress',
        type: 'address'
      }
    ],
    stateMutability: 'nonpayable',
    type: 'constructor'
  },
  {
    inputs: [],
    name: 'ConsumerValid',
    type: 'error'
  },
  {
    inputs: [],
    name: 'FailedToDeposit',
    type: 'error'
  },
  {
    inputs: [],
    name: 'InsufficientBalance',
    type: 'error'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'have',
        type: 'address'
      },
      {
        internalType: 'address',
        name: 'want',
        type: 'address'
      }
    ],
    name: 'OnlyCoordinatorCanFulfill',
    type: 'error'
  },
  {
    inputs: [],
    name: 'OnlyOracle',
    type: 'error'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'oracle',
        type: 'address'
      }
    ],
    name: 'OracleAdded',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'oracle',
        type: 'address'
      }
    ],
    name: 'OracleRemoved',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'address',
        name: 'previousOwner',
        type: 'address'
      },
      {
        indexed: true,
        internalType: 'address',
        name: 'newOwner',
        type: 'address'
      }
    ],
    name: 'OwnershipTransferred',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'uint256',
        name: 'requestId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'uint256',
        name: 'l2RequestId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'address',
        name: 'sender',
        type: 'address'
      },
      {
        indexed: false,
        internalType: 'uint256',
        name: 'callbackGasLimit',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'uint256[]',
        name: 'randomWords',
        type: 'uint256[]'
      }
    ],
    name: 'RandomWordFulfilled',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'uint256',
        name: 'requestId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'address',
        name: 'sender',
        type: 'address'
      }
    ],
    name: 'RandomWordRequested',
    type: 'event'
  },
  {
    inputs: [],
    name: 'REGISTRY',
    outputs: [
      {
        internalType: 'contract IRegistry',
        name: '',
        type: 'address'
      }
    ],
    stateMutability: 'view',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'oracle',
        type: 'address'
      }
    ],
    name: 'addOracle',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [],
    name: 'owner',
    outputs: [
      {
        internalType: 'address',
        name: '',
        type: 'address'
      }
    ],
    stateMutability: 'view',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'uint256',
        name: 'requestId',
        type: 'uint256'
      },
      {
        internalType: 'uint256[]',
        name: 'randomWords',
        type: 'uint256[]'
      }
    ],
    name: 'rawFulfillRandomWords',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'oracle',
        type: 'address'
      }
    ],
    name: 'removeOracle',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'bytes32',
        name: 'keyHash',
        type: 'bytes32'
      },
      {
        internalType: 'uint32',
        name: 'callbackGasLimit',
        type: 'uint32'
      },
      {
        internalType: 'uint32',
        name: 'numWords',
        type: 'uint32'
      },
      {
        internalType: 'uint256',
        name: 'accId',
        type: 'uint256'
      },
      {
        internalType: 'address',
        name: 'sender',
        type: 'address'
      },
      {
        internalType: 'uint256',
        name: 'l2RequestId',
        type: 'uint256'
      }
    ],
    name: 'requestRandomWords',
    outputs: [
      {
        internalType: 'uint256',
        name: '',
        type: 'uint256'
      }
    ],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'newOwner',
        type: 'address'
      }
    ],
    name: 'transferOwnership',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    stateMutability: 'payable',
    type: 'receive'
  }
]
