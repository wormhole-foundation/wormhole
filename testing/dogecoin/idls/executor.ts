/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/executor.json`.
 */
export type Executor = {
  "address": "execXUrAsMnqMmTHj5m7N1YQgsDz3cwGLYCYyuDRciV",
  "metadata": {
    "name": "executor",
    "version": "0.1.0",
    "spec": "0.1.0",
    "description": "Created with Anchor"
  },
  "instructions": [
    {
      "name": "requestForExecution",
      "discriminator": [
        109,
        107,
        87,
        37,
        151,
        192,
        119,
        115
      ],
      "accounts": [
        {
          "name": "payer",
          "writable": true,
          "signer": true
        },
        {
          "name": "payee",
          "writable": true
        },
        {
          "name": "systemProgram",
          "address": "11111111111111111111111111111111"
        }
      ],
      "args": [
        {
          "name": "args",
          "type": {
            "defined": {
              "name": "requestForExecutionArgs"
            }
          }
        }
      ]
    }
  ],
  "errors": [
    {
      "code": 6000,
      "name": "invalidArguments",
      "msg": "invalidArguments"
    },
    {
      "code": 6001,
      "name": "quoteSrcChainMismatch",
      "msg": "quoteSrcChainMismatch"
    },
    {
      "code": 6002,
      "name": "quoteDstChainMismatch",
      "msg": "quoteDstChainMismatch"
    },
    {
      "code": 6003,
      "name": "quoteExpired",
      "msg": "quoteExpired"
    },
    {
      "code": 6004,
      "name": "quotePayeeMismatch",
      "msg": "quotePayeeMismatch"
    }
  ],
  "types": [
    {
      "name": "requestForExecutionArgs",
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "amount",
            "type": "u64"
          },
          {
            "name": "dstChain",
            "type": "u16"
          },
          {
            "name": "dstAddr",
            "type": {
              "array": [
                "u8",
                32
              ]
            }
          },
          {
            "name": "refundAddr",
            "type": "pubkey"
          },
          {
            "name": "signedQuoteBytes",
            "type": "bytes"
          },
          {
            "name": "requestBytes",
            "type": "bytes"
          },
          {
            "name": "relayInstructions",
            "type": "bytes"
          }
        ]
      }
    }
  ]
};
