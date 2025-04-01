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
    "version": "0.0.0",
    "spec": "0.1.0",
    "description": "Anchor Interface for Wormhole Verify VAA Shim"
  },
  "instructions": [
    {
      "name": "closeSignatures",
      "docs": [
        "Allows the initial payer to close the signature account, reclaiming the",
        "rent taken by the post signatures instruction."
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
        "Creates or appends to a guardian signatures account for subsequent use",
        "by the verify hash instruction.",
        "",
        "This instruction is necessary due to the Wormhole VAA body, which has an",
        "arbitrary size, and 13 guardian signatures (a quorum of the current 19",
        "mainnet guardians, 66 bytes each) alongside the required accounts is",
        "likely larger than the transaction size limit on Solana (1232 bytes).",
        "",
        "This instruction will also allow for the verification of other messages",
        "which guardians sign, such as query results.",
        "",
        "This instruction allows for the initial payer to append additional",
        "signatures to the account by calling the instruction again. Subsequent",
        "calls may be necessary if a quorum of signatures from the current guardian",
        "set grows larger than can fit into a single transaction.",
        "",
        "The guardian signatures account can be closed by the initial payer via",
        "the close signatures instruction, which will refund this payer."
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
      "name": "verifyHash",
      "docs": [
        "This instruction is intended to be invoked via CPI call. It verifies a",
        "digest against a guardian signatures account and a Wormhole Core Bridge",
        "guardian set account.",
        "",
        "Prior to this call (and likely in a separate transaction), call the post",
        "signatures instruction to create the guardian signatures account.",
        "",
        "Immediately after this verify call, call the close signatures",
        "instruction to reclaim the rent paid to create the guardian signatures",
        "account.",
        "",
        "A v1 VAA digest can be computed as follows:",
        "```rust",
        "use wormhole_svm_definitions::compute_keccak_digest;",
        "",
        "// `vec_body` is the encoded body of the VAA.",
        "# let vaa_body = vec![];",
        "let digest = compute_keccak_digest(",
        "solana_program::keccak::hash(&vaa_body),",
        "None, // there is no prefix for V1 messages",
        ");",
        "```",
        "",
        "A QueryResponse digest can be computed as follows:",
        "```rust",
        "# mod wormhole_query_sdk {",
        "#    pub const MESSAGE_PREFIX: &'static [u8] = b\"ruh roh\";",
        "# }",
        "use wormhole_query_sdk::MESSAGE_PREFIX;",
        "use wormhole_svm_definitions::compute_keccak_digest;",
        "",
        "# let query_response_bytes = vec![];",
        "let digest = compute_keccak_digest(",
        "solana_program::keccak::hash(&query_response_bytes),",
        "Some(MESSAGE_PREFIX)",
        ");",
        "```"
      ],
      "discriminator": [
        22,
        152,
        160,
        69,
        241,
        148,
        14,
        124
      ],
      "accounts": [
        {
          "name": "guardianSet",
          "docs": [
            "Guardian set used for signature verification."
          ]
        },
        {
          "name": "guardianSignatures",
          "docs": [
            "Stores unverified guardian signatures as they are too large to fit in",
            "the instruction data."
          ]
        }
      ],
      "args": [
        {
          "name": "guardianSetBump",
          "type": "u8"
        },
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
    }
  ]
};

