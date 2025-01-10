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
      "docs": [
        "Allows the initial payer to close the signature account, reclaiming the rent taken by `post_signatures`."
      ],
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
      "docs": [
        "Creates or appends to a GuardianSignatures account for subsequent use by verify_vaa.",
        "This is necessary as the Wormhole VAA body, which has an arbitrary size,",
        "and 13 guardian signatures (a quorum of the current 19 mainnet guardians, 66 bytes each)",
        "alongside the required accounts is likely larger than the transaction size limit on Solana (1232 bytes).",
        "This will also allow for the verification of other messages which guardians sign, such as QueryResults.",
        "",
        "This instruction allows for the initial payer to append additional signatures to the account by calling the instruction again.",
        "This may be necessary if a quorum of signatures from the current guardian set grows larger than can fit into a single transaction.",
        "",
        "The GuardianSignatures account can be closed by the initial payer via close_signatures, which will refund the initial payer."
      ],
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
      "docs": [
        "This instruction is intended to be invoked via CPI call. It verifies a digest against a GuardianSignatures account",
        "and a core bridge GuardianSet.",
        "Prior to this call, and likely in a separate transaction, `post_signatures` must be called to create the account.",
        "Immediately after this call, `close_signatures` should be called to reclaim the lamports.",
        "",
        "A v1 VAA digest can be computed as follows:",
        "```rust",
        "let message_hash = &solana_program::keccak::hashv(&[&vaa_body]).to_bytes();",
        "let digest = keccak::hash(message_hash.as_slice()).to_bytes();",
        "```",
        "",
        "A QueryResponse digest can be computed as follows:",
        "```rust",
        "use wormhole_query_sdk::MESSAGE_PREFIX;",
        "let message_hash = [",
        "MESSAGE_PREFIX,",
        "&solana_program::keccak::hashv(&[&bytes]).to_bytes(),",
        "]",
        ".concat();",
        "let digest = keccak::hash(message_hash.as_slice()).to_bytes();",
        "```"
      ],
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
