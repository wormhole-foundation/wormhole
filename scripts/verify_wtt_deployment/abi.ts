export const TOKEN_BRIDGE_ABI = [
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        internalType: 'address',
        name: 'previousAdmin',
        type: 'address',
      },
      {
        indexed: false,
        internalType: 'address',
        name: 'newAdmin',
        type: 'address',
      },
    ],
    name: 'AdminChanged',
    type: 'event',
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'address',
        name: 'beacon',
        type: 'address',
      },
    ],
    name: 'BeaconUpgraded',
    type: 'event',
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'address',
        name: 'oldContract',
        type: 'address',
      },
      {
        indexed: true,
        internalType: 'address',
        name: 'newContract',
        type: 'address',
      },
    ],
    name: 'ContractUpgraded',
    type: 'event',
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'uint16',
        name: 'emitterChainId',
        type: 'uint16',
      },
      {
        indexed: true,
        internalType: 'bytes32',
        name: 'emitterAddress',
        type: 'bytes32',
      },
      {
        indexed: true,
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    name: 'TransferRedeemed',
    type: 'event',
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'address',
        name: 'implementation',
        type: 'address',
      },
    ],
    name: 'Upgraded',
    type: 'event',
  },
  {
    inputs: [],
    name: 'WETH',
    outputs: [
      {
        internalType: 'contract IWETH',
        name: '',
        type: 'address',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: '_parseTransferCommon',
    outputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'amount',
            type: 'uint256',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'to',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'toChain',
            type: 'uint16',
          },
          {
            internalType: 'uint256',
            name: 'fee',
            type: 'uint256',
          },
        ],
        internalType: 'struct BridgeStructs.Transfer',
        name: 'transfer',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'tokenAddress',
        type: 'address',
      },
      {
        internalType: 'uint32',
        name: 'nonce',
        type: 'uint32',
      },
    ],
    name: 'attestToken',
    outputs: [
      {
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    stateMutability: 'payable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'uint16',
        name: 'chainId_',
        type: 'uint16',
      },
    ],
    name: 'bridgeContracts',
    outputs: [
      {
        internalType: 'bytes32',
        name: '',
        type: 'bytes32',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'chainId',
    outputs: [
      {
        internalType: 'uint16',
        name: '',
        type: 'uint16',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'completeTransfer',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'completeTransferAndUnwrapETH',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'completeTransferAndUnwrapETHWithPayload',
    outputs: [
      {
        internalType: 'bytes',
        name: '',
        type: 'bytes',
      },
    ],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'completeTransferWithPayload',
    outputs: [
      {
        internalType: 'bytes',
        name: '',
        type: 'bytes',
      },
    ],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'createWrapped',
    outputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
    ],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'uint8',
            name: 'decimals',
            type: 'uint8',
          },
          {
            internalType: 'bytes32',
            name: 'symbol',
            type: 'bytes32',
          },
          {
            internalType: 'bytes32',
            name: 'name',
            type: 'bytes32',
          },
        ],
        internalType: 'struct BridgeStructs.AssetMeta',
        name: 'meta',
        type: 'tuple',
      },
    ],
    name: 'encodeAssetMeta',
    outputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'amount',
            type: 'uint256',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'to',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'toChain',
            type: 'uint16',
          },
          {
            internalType: 'uint256',
            name: 'fee',
            type: 'uint256',
          },
        ],
        internalType: 'struct BridgeStructs.Transfer',
        name: 'transfer',
        type: 'tuple',
      },
    ],
    name: 'encodeTransfer',
    outputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'amount',
            type: 'uint256',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'to',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'toChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'fromAddress',
            type: 'bytes32',
          },
          {
            internalType: 'bytes',
            name: 'payload',
            type: 'bytes',
          },
        ],
        internalType: 'struct BridgeStructs.TransferWithPayload',
        name: 'transfer',
        type: 'tuple',
      },
    ],
    name: 'encodeTransferWithPayload',
    outputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [],
    name: 'evmChainId',
    outputs: [
      {
        internalType: 'uint256',
        name: '',
        type: 'uint256',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'finality',
    outputs: [
      {
        internalType: 'uint8',
        name: '',
        type: 'uint8',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes32',
        name: 'hash',
        type: 'bytes32',
      },
    ],
    name: 'governanceActionIsConsumed',
    outputs: [
      {
        internalType: 'bool',
        name: '',
        type: 'bool',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'governanceChainId',
    outputs: [
      {
        internalType: 'uint16',
        name: '',
        type: 'uint16',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'governanceContract',
    outputs: [
      {
        internalType: 'bytes32',
        name: '',
        type: 'bytes32',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'isFork',
    outputs: [
      {
        internalType: 'bool',
        name: '',
        type: 'bool',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'impl',
        type: 'address',
      },
    ],
    name: 'isInitialized',
    outputs: [
      {
        internalType: 'bool',
        name: '',
        type: 'bool',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes32',
        name: 'hash',
        type: 'bytes32',
      },
    ],
    name: 'isTransferCompleted',
    outputs: [
      {
        internalType: 'bool',
        name: '',
        type: 'bool',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
    ],
    name: 'isWrappedAsset',
    outputs: [
      {
        internalType: 'bool',
        name: '',
        type: 'bool',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
    ],
    name: 'outstandingBridged',
    outputs: [
      {
        internalType: 'uint256',
        name: '',
        type: 'uint256',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parseAssetMeta',
    outputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'uint8',
            name: 'decimals',
            type: 'uint8',
          },
          {
            internalType: 'bytes32',
            name: 'symbol',
            type: 'bytes32',
          },
          {
            internalType: 'bytes32',
            name: 'name',
            type: 'bytes32',
          },
        ],
        internalType: 'struct BridgeStructs.AssetMeta',
        name: 'meta',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parsePayloadID',
    outputs: [
      {
        internalType: 'uint8',
        name: 'payloadID',
        type: 'uint8',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedRecoverChainId',
        type: 'bytes',
      },
    ],
    name: 'parseRecoverChainId',
    outputs: [
      {
        components: [
          {
            internalType: 'bytes32',
            name: 'module',
            type: 'bytes32',
          },
          {
            internalType: 'uint8',
            name: 'action',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'evmChainId',
            type: 'uint256',
          },
          {
            internalType: 'uint16',
            name: 'newChainId',
            type: 'uint16',
          },
        ],
        internalType: 'struct BridgeStructs.RecoverChainId',
        name: 'rci',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parseRegisterChain',
    outputs: [
      {
        components: [
          {
            internalType: 'bytes32',
            name: 'module',
            type: 'bytes32',
          },
          {
            internalType: 'uint8',
            name: 'action',
            type: 'uint8',
          },
          {
            internalType: 'uint16',
            name: 'chainId',
            type: 'uint16',
          },
          {
            internalType: 'uint16',
            name: 'emitterChainID',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'emitterAddress',
            type: 'bytes32',
          },
        ],
        internalType: 'struct BridgeStructs.RegisterChain',
        name: 'chain',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parseTransfer',
    outputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'amount',
            type: 'uint256',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'to',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'toChain',
            type: 'uint16',
          },
          {
            internalType: 'uint256',
            name: 'fee',
            type: 'uint256',
          },
        ],
        internalType: 'struct BridgeStructs.Transfer',
        name: 'transfer',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parseTransferWithPayload',
    outputs: [
      {
        components: [
          {
            internalType: 'uint8',
            name: 'payloadID',
            type: 'uint8',
          },
          {
            internalType: 'uint256',
            name: 'amount',
            type: 'uint256',
          },
          {
            internalType: 'bytes32',
            name: 'tokenAddress',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'tokenChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'to',
            type: 'bytes32',
          },
          {
            internalType: 'uint16',
            name: 'toChain',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'fromAddress',
            type: 'bytes32',
          },
          {
            internalType: 'bytes',
            name: 'payload',
            type: 'bytes',
          },
        ],
        internalType: 'struct BridgeStructs.TransferWithPayload',
        name: 'transfer',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encoded',
        type: 'bytes',
      },
    ],
    name: 'parseUpgrade',
    outputs: [
      {
        components: [
          {
            internalType: 'bytes32',
            name: 'module',
            type: 'bytes32',
          },
          {
            internalType: 'uint8',
            name: 'action',
            type: 'uint8',
          },
          {
            internalType: 'uint16',
            name: 'chainId',
            type: 'uint16',
          },
          {
            internalType: 'bytes32',
            name: 'newContract',
            type: 'bytes32',
          },
        ],
        internalType: 'struct BridgeStructs.UpgradeContract',
        name: 'chain',
        type: 'tuple',
      },
    ],
    stateMutability: 'pure',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVM',
        type: 'bytes',
      },
    ],
    name: 'registerChain',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVM',
        type: 'bytes',
      },
    ],
    name: 'submitRecoverChainId',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [],
    name: 'tokenImplementation',
    outputs: [
      {
        internalType: 'address',
        name: '',
        type: 'address',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
      {
        internalType: 'uint256',
        name: 'amount',
        type: 'uint256',
      },
      {
        internalType: 'uint16',
        name: 'recipientChain',
        type: 'uint16',
      },
      {
        internalType: 'bytes32',
        name: 'recipient',
        type: 'bytes32',
      },
      {
        internalType: 'uint256',
        name: 'arbiterFee',
        type: 'uint256',
      },
      {
        internalType: 'uint32',
        name: 'nonce',
        type: 'uint32',
      },
    ],
    name: 'transferTokens',
    outputs: [
      {
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    stateMutability: 'payable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
      {
        internalType: 'uint256',
        name: 'amount',
        type: 'uint256',
      },
      {
        internalType: 'uint16',
        name: 'recipientChain',
        type: 'uint16',
      },
      {
        internalType: 'bytes32',
        name: 'recipient',
        type: 'bytes32',
      },
      {
        internalType: 'uint32',
        name: 'nonce',
        type: 'uint32',
      },
      {
        internalType: 'bytes',
        name: 'payload',
        type: 'bytes',
      },
    ],
    name: 'transferTokensWithPayload',
    outputs: [
      {
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    stateMutability: 'payable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVm',
        type: 'bytes',
      },
    ],
    name: 'updateWrapped',
    outputs: [
      {
        internalType: 'address',
        name: 'token',
        type: 'address',
      },
    ],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'bytes',
        name: 'encodedVM',
        type: 'bytes',
      },
    ],
    name: 'upgrade',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [],
    name: 'wormhole',
    outputs: [
      {
        internalType: 'contract IWormhole',
        name: '',
        type: 'address',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'uint16',
        name: 'recipientChain',
        type: 'uint16',
      },
      {
        internalType: 'bytes32',
        name: 'recipient',
        type: 'bytes32',
      },
      {
        internalType: 'uint256',
        name: 'arbiterFee',
        type: 'uint256',
      },
      {
        internalType: 'uint32',
        name: 'nonce',
        type: 'uint32',
      },
    ],
    name: 'wrapAndTransferETH',
    outputs: [
      {
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    stateMutability: 'payable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'uint16',
        name: 'recipientChain',
        type: 'uint16',
      },
      {
        internalType: 'bytes32',
        name: 'recipient',
        type: 'bytes32',
      },
      {
        internalType: 'uint32',
        name: 'nonce',
        type: 'uint32',
      },
      {
        internalType: 'bytes',
        name: 'payload',
        type: 'bytes',
      },
    ],
    name: 'wrapAndTransferETHWithPayload',
    outputs: [
      {
        internalType: 'uint64',
        name: 'sequence',
        type: 'uint64',
      },
    ],
    stateMutability: 'payable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'uint16',
        name: 'tokenChainId',
        type: 'uint16',
      },
      {
        internalType: 'bytes32',
        name: 'tokenAddress',
        type: 'bytes32',
      },
    ],
    name: 'wrappedAsset',
    outputs: [
      {
        internalType: 'address',
        name: '',
        type: 'address',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    stateMutability: 'payable',
    type: 'receive',
  },
]