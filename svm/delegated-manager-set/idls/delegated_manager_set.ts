/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/delegated_manager_set.json`.
 */
export type DelegatedManagerSet = {
  "address": "wdmsTJP6YnsfeQjPuuEzGCrHmZvTmNy8VkxMCK8JkBX",
  "metadata": {
    "name": "delegatedManagerSet",
    "version": "0.1.0",
    "spec": "0.1.0",
    "description": "Created with Anchor"
  },
  "instructions": [
    {
      "name": "submitNewManagerSet",
      "discriminator": [
        19,
        18,
        149,
        235,
        253,
        47,
        60,
        58
      ],
      "accounts": [
        {
          "name": "payer",
          "writable": true,
          "signer": true
        },
        {
          "name": "guardianSet",
          "docs": [
            "Derivation is checked by the shim."
          ]
        },
        {
          "name": "guardianSignatures",
          "docs": [
            "Ownership ownership and discriminator is checked by the shim."
          ]
        },
        {
          "name": "consumed",
          "docs": [
            "The derivation is confirmed in the instruction"
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  99,
                  111,
                  110,
                  115,
                  117,
                  109,
                  101,
                  100,
                  95,
                  118,
                  97,
                  97
                ]
              },
              {
                "kind": "arg",
                "path": "args.digest"
              }
            ]
          }
        },
        {
          "name": "managerSetIndex",
          "docs": [
            "Stores the current manager set index for this manager chain.",
            "Initialized on first manager set update, or updated on subsequent ones."
          ],
          "writable": true
        },
        {
          "name": "managerSet",
          "docs": [
            "Stores the new manager set."
          ],
          "writable": true
        },
        {
          "name": "wormholeVerifyVaaShim",
          "address": "EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at"
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
              "name": "submitNewManagerSetArgs"
            }
          }
        }
      ]
    }
  ],
  "accounts": [
    {
      "name": "managerSet",
      "discriminator": [
        188,
        16,
        135,
        64,
        103,
        222,
        63,
        182
      ]
    },
    {
      "name": "managerSetIndex",
      "discriminator": [
        42,
        93,
        41,
        30,
        20,
        230,
        157,
        75
      ]
    }
  ],
  "errors": [
    {
      "code": 6000,
      "name": "digestMismatch",
      "msg": "Digest argument does not match computed digest from VAA body"
    },
    {
      "code": 6001,
      "name": "invalidVaaBody",
      "msg": "Failed to parse VAA body"
    },
    {
      "code": 6002,
      "name": "invalidGovernanceChain",
      "msg": "VAA is not from the governance chain"
    },
    {
      "code": 6003,
      "name": "invalidGovernanceEmitter",
      "msg": "VAA is not from the governance emitter"
    },
    {
      "code": 6004,
      "name": "governancePayloadTooShort",
      "msg": "Governance payload too short"
    },
    {
      "code": 6005,
      "name": "invalidGovernanceModule",
      "msg": "Invalid governance module"
    },
    {
      "code": 6006,
      "name": "invalidGovernanceAction",
      "msg": "Invalid governance action"
    },
    {
      "code": 6007,
      "name": "invalidTargetChain",
      "msg": "Invalid target chain"
    },
    {
      "code": 6008,
      "name": "invalidManagerSetIndex",
      "msg": "Manager set index must increment by 1"
    }
  ],
  "types": [
    {
      "name": "managerSet",
      "docs": [
        "Stores a manager set for a given chain ID and index.",
        "PDA seeds: [\"manager_set\", manager_chain_id, manager_set_index]"
      ],
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "managerChainId",
            "docs": [
              "The manager chain ID this set is for."
            ],
            "type": "u16"
          },
          {
            "name": "index",
            "docs": [
              "The manager set index."
            ],
            "type": "u32"
          },
          {
            "name": "managerSet",
            "docs": [
              "The raw manager set bytes."
            ],
            "type": "bytes"
          }
        ]
      }
    },
    {
      "name": "managerSetIndex",
      "docs": [
        "Stores the current manager set index for a given chain ID.",
        "PDA seeds: [\"manager_set_index\", manager_chain_id]"
      ],
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "managerChainId",
            "docs": [
              "The manager chain ID this index is for."
            ],
            "type": "u16"
          },
          {
            "name": "currentIndex",
            "docs": [
              "The current manager set index."
            ],
            "type": "u32"
          }
        ]
      }
    },
    {
      "name": "submitNewManagerSetArgs",
      "type": {
        "kind": "struct",
        "fields": [
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
          },
          {
            "name": "vaaBody",
            "type": "bytes"
          }
        ]
      }
    }
  ]
};
