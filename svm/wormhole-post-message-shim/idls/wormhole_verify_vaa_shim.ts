/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/wormhole_verify_vaa_shim.json`.
 */
export type WormholeVerifyVaaShim = {
  "address": "EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at",
  "metadata": {
    "name": "wormholeVerifyVaaShim",
    "version": "0.1.0",
    "spec": "0.1.0",
    "description": "Created with Anchor"
  },
  "instructions": [
    {
      "name": "closeSignatures",
      "discriminator": [
        192,
        65,
        63,
        117,
        213,
        138,
        179,
        190
      ],
      "accounts": [
        {
          "name": "guardianSignatures",
          "writable": true
        },
        {
          "name": "refundRecipient",
          "writable": true,
          "signer": true,
          "relations": [
            "guardianSignatures"
          ]
        }
      ],
      "args": []
    },
    {
      "name": "postSignatures",
      "discriminator": [
        138,
        2,
        53,
        166,
        45,
        77,
        137,
        51
      ],
      "accounts": [
        {
          "name": "payer",
          "writable": true,
          "signer": true
        },
        {
          "name": "guardianSignatures",
          "writable": true,
          "signer": true
        },
        {
          "name": "systemProgram",
          "address": "11111111111111111111111111111111"
        }
      ],
      "args": [
        {
          "name": "guardianSetIndex",
          "type": "u32"
        },
        {
          "name": "totalSignatures",
          "type": "u8"
        },
        {
          "name": "guardianSignatures",
          "type": {
            "vec": {
              "array": [
                "u8",
                66
              ]
            }
          }
        }
      ]
    },
    {
      "name": "verifyVaa",
      "discriminator": [
        147,
        254,
        88,
        41,
        24,
        223,
        219,
        29
      ],
      "accounts": [
        {
          "name": "guardianSet",
          "docs": [
            "Guardian set used for signature verification."
          ],
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  71,
                  117,
                  97,
                  114,
                  100,
                  105,
                  97,
                  110,
                  83,
                  101,
                  116
                ]
              },
              {
                "kind": "account",
                "path": "guardian_signatures.guardian_set_index_be",
                "account": "guardianSignatures"
              }
            ],
            "program": {
              "kind": "const",
              "value": [
                14,
                10,
                88,
                154,
                65,
                165,
                95,
                189,
                102,
                197,
                42,
                71,
                95,
                45,
                146,
                166,
                211,
                220,
                155,
                71,
                71,
                17,
                76,
                185,
                175,
                130,
                90,
                152,
                181,
                69,
                211,
                206
              ]
            }
          }
        },
        {
          "name": "guardianSignatures",
          "docs": [
            "Stores unverified guardian signatures as they are too large to fit in the instruction data."
          ]
        }
      ],
      "args": [
        {
          "name": "digest",
          "type": {
            "array": [
              "u8",
              32
            ]
          }
        }
      ]
    }
  ],
  "accounts": [
    {
      "name": "guardianSignatures",
      "discriminator": [
        203,
        184,
        130,
        157,
        113,
        14,
        184,
        83
      ]
    },
    {
      "name": "wormholeGuardianSet",
      "discriminator": [
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0
      ]
    }
  ],
  "errors": [
    {
      "code": 6000,
      "name": "emptyGuardianSignatures",
      "msg": "emptyGuardianSignatures"
    },
    {
      "code": 6001,
      "name": "writeAuthorityMismatch",
      "msg": "writeAuthorityMismatch"
    },
    {
      "code": 6002,
      "name": "guardianSetExpired",
      "msg": "guardianSetExpired"
    },
    {
      "code": 6003,
      "name": "noQuorum",
      "msg": "noQuorum"
    },
    {
      "code": 6004,
      "name": "invalidSignature",
      "msg": "invalidSignature"
    },
    {
      "code": 6005,
      "name": "invalidGuardianIndexNonIncreasing",
      "msg": "invalidGuardianIndexNonIncreasing"
    },
    {
      "code": 6006,
      "name": "invalidGuardianIndexOutOfRange",
      "msg": "invalidGuardianIndexOutOfRange"
    },
    {
      "code": 6007,
      "name": "invalidGuardianKeyRecovery",
      "msg": "invalidGuardianKeyRecovery"
    }
  ],
  "types": [
    {
      "name": "guardianSignatures",
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "refundRecipient",
            "docs": [
              "Payer of this guardian signatures account.",
              "Only they may amend signatures.",
              "Used for reimbursements upon cleanup."
            ],
            "type": "pubkey"
          },
          {
            "name": "guardianSetIndexBe",
            "docs": [
              "Guardian set index that these signatures correspond to.",
              "Storing this simplifies the integrator data.",
              "Using big-endian to match the derivation used by the core bridge."
            ],
            "type": {
              "array": [
                "u8",
                4
              ]
            }
          },
          {
            "name": "guardianSignatures",
            "docs": [
              "Unverified guardian signatures."
            ],
            "type": {
              "vec": {
                "array": [
                  "u8",
                  66
                ]
              }
            }
          }
        ]
      }
    },
    {
      "name": "wormholeGuardianSet",
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "index",
            "docs": [
              "Index representing an incrementing version number for this guardian set."
            ],
            "type": "u32"
          },
          {
            "name": "keys",
            "docs": [
              "Ethereum-style public keys."
            ],
            "type": {
              "vec": {
                "array": [
                  "u8",
                  20
                ]
              }
            }
          },
          {
            "name": "creationTime",
            "docs": [
              "Timestamp representing the time this guardian became active."
            ],
            "type": "u32"
          },
          {
            "name": "expirationTime",
            "docs": [
              "Expiration time when VAAs issued by this set are no longer valid."
            ],
            "type": "u32"
          }
        ]
      }
    }
  ]
};
