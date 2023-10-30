export type Wormhole = {
  version: '0.1.0';
  name: 'wormhole';
  instructions: [
    {
      name: 'initialize';
      accounts: [
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'guardianSet';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'feeCollector';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'clock';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [
        {
          name: 'guardianSetExpirationTime';
          type: 'u32';
        },
        {
          name: 'fee';
          type: 'u64';
        },
        {
          name: 'initialGuardians';
          type: {
            vec: {
              array: ['u8', 20];
            };
          };
        },
      ];
    },
    {
      name: 'postMessage';
      accounts: [
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'message';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'emitter';
          isMut: false;
          isSigner: true;
        },
        {
          name: 'sequence';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'feeCollector';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'clock';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [
        {
          name: 'nonce';
          type: 'u32';
        },
        {
          name: 'payload';
          type: 'bytes';
        },
        {
          name: 'consistencyLevel';
          type: 'u8';
        },
      ];
    },
    {
      name: 'postVaa';
      accounts: [
        {
          name: 'guardianSet';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'bridge';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'signatureSet';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'vaa';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'clock';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [
        {
          name: 'version';
          type: 'u8';
        },
        {
          name: 'guardianSetIndex';
          type: 'u32';
        },
        {
          name: 'timestamp';
          type: 'u32';
        },
        {
          name: 'nonce';
          type: 'u32';
        },
        {
          name: 'emitterChain';
          type: 'u16';
        },
        {
          name: 'emitterAddress';
          type: {
            array: ['u8', 32];
          };
        },
        {
          name: 'sequence';
          type: 'u64';
        },
        {
          name: 'consistencyLevel';
          type: 'u8';
        },
        {
          name: 'payload';
          type: 'bytes';
        },
      ];
    },
    {
      name: 'setFees';
      accounts: [
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'vaa';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'claim';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [];
    },
    {
      name: 'transferFees';
      accounts: [
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'vaa';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'claim';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'feeCollector';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'recipient';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [];
    },
    {
      name: 'upgradeContract';
      accounts: [
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'vaa';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'claim';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'upgradeAuthority';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'spill';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'implementation';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'programData';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'wormholeProgram';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'clock';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'bpfLoaderUpgradeable';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [];
    },
    {
      name: 'upgradeGuardianSet';
      accounts: [
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'vaa';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'claim';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'guardianSetOld';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'guardianSetNew';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [];
    },
    {
      name: 'verifySignatures';
      accounts: [
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'guardianSet';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'signatureSet';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'instructions';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [
        {
          name: 'signatureStatus';
          type: {
            array: ['i8', 19];
          };
        },
      ];
    },
    {
      name: 'postMessageUnreliable';
      accounts: [
        {
          name: 'bridge';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'message';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'emitter';
          isMut: false;
          isSigner: true;
        },
        {
          name: 'sequence';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'payer';
          isMut: true;
          isSigner: true;
        },
        {
          name: 'feeCollector';
          isMut: true;
          isSigner: false;
        },
        {
          name: 'clock';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'rent';
          isMut: false;
          isSigner: false;
        },
        {
          name: 'systemProgram';
          isMut: false;
          isSigner: false;
        },
      ];
      args: [
        {
          name: 'nonce';
          type: 'u32';
        },
        {
          name: 'payload';
          type: 'bytes';
        },
        {
          name: 'consistencyLevel';
          type: 'u8';
        },
      ];
    },
  ];
  accounts: [
    {
      name: 'PostedMessage';
      type: {
        kind: 'struct';
        fields: [
          {
            name: 'vaaVersion';
            type: 'u8';
          },
          {
            name: 'consistencyLevel';
            type: 'u8';
          },
          {
            name: 'vaaTime';
            type: 'u32';
          },
          {
            name: 'vaaSignatureAccount';
            type: 'publicKey';
          },
          {
            name: 'submissionTime';
            type: 'u32';
          },
          {
            name: 'nonce';
            type: 'u32';
          },
          {
            name: 'sequence';
            type: 'u64';
          },
          {
            name: 'emitterChain';
            type: 'u16';
          },
          {
            name: 'emitterAddress';
            type: {
              array: ['u8', 32];
            };
          },
          {
            name: 'payload';
            type: 'bytes';
          },
        ];
      };
    },
    {
      name: 'PostedVAA';
      type: {
        kind: 'struct';
        fields: [
          {
            name: 'vaaVersion';
            type: 'u8';
          },
          {
            name: 'consistencyLevel';
            type: 'u8';
          },
          {
            name: 'vaaTime';
            type: 'u32';
          },
          {
            name: 'vaaSignatureAccount';
            type: 'publicKey';
          },
          {
            name: 'submissionTime';
            type: 'u32';
          },
          {
            name: 'nonce';
            type: 'u32';
          },
          {
            name: 'sequence';
            type: 'u64';
          },
          {
            name: 'emitterChain';
            type: 'u16';
          },
          {
            name: 'emitterAddress';
            type: {
              array: ['u8', 32];
            };
          },
          {
            name: 'payload';
            type: 'bytes';
          },
        ];
      };
    },
  ];
};
