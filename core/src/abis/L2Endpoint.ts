export const L2EndpointAbis = [
  {
    inputs: [
      {
        internalType: 'address',
        name: 'aggregator',
        type: 'address'
      }
    ],
    name: 'InvalidAggregator',
    type: 'error'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'submitter',
        type: 'address'
      }
    ],
    name: 'InvalidSubmitter',
    type: 'error'
  },
  {
    inputs: [],
    name: 'Reentrant',
    type: 'error'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'newAggregator',
        type: 'address'
      }
    ],
    name: 'AggregatorAdded',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'newAggregator',
        type: 'address'
      }
    ],
    name: 'AggregatorRemoved',
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
        indexed: true,
        internalType: 'uint256',
        name: 'requestId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'uint256[]',
        name: 'randomWords',
        type: 'uint256[]'
      },
      {
        indexed: false,
        internalType: 'bool',
        name: 'success',
        type: 'bool'
      }
    ],
    name: 'RandomWordsFulfilled',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'bytes32',
        name: 'keyHash',
        type: 'bytes32'
      },
      {
        indexed: false,
        internalType: 'uint256',
        name: 'requestId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'uint256',
        name: 'preSeed',
        type: 'uint256'
      },
      {
        indexed: true,
        internalType: 'uint64',
        name: 'accId',
        type: 'uint64'
      },
      {
        indexed: false,
        internalType: 'uint32',
        name: 'callbackGasLimit',
        type: 'uint32'
      },
      {
        indexed: false,
        internalType: 'uint32',
        name: 'numWords',
        type: 'uint32'
      },
      {
        indexed: true,
        internalType: 'address',
        name: 'sender',
        type: 'address'
      }
    ],
    name: 'RandomWordsRequested',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'uint256',
        name: 'roundId',
        type: 'uint256'
      },
      {
        indexed: false,
        internalType: 'int256',
        name: 'submission',
        type: 'int256'
      }
    ],
    name: 'Submitted',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'newSubmitter',
        type: 'address'
      }
    ],
    name: 'SubmitterAdded',
    type: 'event'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'newSubmitter',
        type: 'address'
      }
    ],
    name: 'SubmitterRemoved',
    type: 'event'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: '_newAggregator',
        type: 'address'
      }
    ],
    name: 'addAggregator',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: '_newSubmitter',
        type: 'address'
      }
    ],
    name: 'addSubmitter',
    outputs: [],
    stateMutability: 'nonpayable',
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
    name: 'fulfillRandomWords',
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
        internalType: 'address',
        name: '_aggregator',
        type: 'address'
      }
    ],
    name: 'removeAggregator',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: '_submitter',
        type: 'address'
      }
    ],
    name: 'removeSubmitter',
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
        internalType: 'uint64',
        name: 'accId',
        type: 'uint64'
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
    inputs: [],
    name: 'sAggregatorCount',
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
    name: 'sSubmitterCount',
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
    inputs: [
      {
        internalType: 'uint256',
        name: '_roundId',
        type: 'uint256'
      },
      {
        internalType: 'int256',
        name: '_submission',
        type: 'int256'
      },
      {
        internalType: 'address',
        name: '_aggregator',
        type: 'address'
      }
    ],
    name: 'submit',
    outputs: [],
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
  }
]
