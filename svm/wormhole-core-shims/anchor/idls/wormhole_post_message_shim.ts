/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/wormhole_post_message_shim.json`.
 */
export type WormholePostMessageShim = {
  "address": "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
  "metadata": {
    "name": "wormholePostMessageShim",
    "version": "0.0.0",
    "spec": "0.1.0",
    "description": "Anchor Interface for Wormhole Post Message Shim"
  },
  "instructions": [
    {
      "name": "postMessage",
      "docs": [
        "This instruction is intended to be a significantly cheaper alternative",
        "to the post message instruction on Wormhole Core Bridge program. It",
        "achieves this by reusing the message account (per emitter) via the post",
        "message unreliable instruction and emitting data via self-CPI (Anchor",
        "event) for the guardian to observe. This instruction data contains",
        "information previously found only in the resulting message account.",
        "",
        "Because this instruction passes through the emitter and calls the post",
        "message unreliable instruction on the Wormhole Core Bridge, it can be",
        "used without disruption.",
        "",
        "NOTE: In the initial message publication for a new emitter, this will",
        "require one additional CPI call depth when compared to using the",
        "Wormhole Core Bridge directly. If this initial call depth is an issue,",
        "emit an empty message on initialization (or migration) in order to",
        "instantiate the message account. Posting a message will result in a VAA",
        "from your emitter, so be careful to avoid any issues that may result",
        "from this first message.",
        "",
        "Call depth of direct case:",
        "1. post message (Wormhole Post Message Shim)",
        "2. multiple CPI",
        "- post message unreliable (Wormhole Core Bridge)",
        "- Anchor event of `MesssageEvent` (Wormhole Post Message Shim)",
        "",
        "Call depth of integrator case:",
        "1. integrator instruction",
        "2. CPI post message (Wormhole Post Message Shim)",
        "3. multiple CPI",
        "- post message unreliable (Wormhole Core Bridge)",
        "- Anchor event of `MesssageEvent` (Wormhole Post Message Shim)"
      ],
      "discriminator": [
        214,
        50,
        100,
        209,
        38,
        34,
        7,
        76
      ],
      "accounts": [
        {
          "name": "bridge",
          "docs": [
            "Wormhole Core Bridge config. The Wormhole Core Bridge program's post",
            "message instruction requires this account to be mutable."
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  66,
                  114,
                  105,
                  100,
                  103,
                  101
                ]
              }
            ],
            "program": {
              "kind": "account",
              "path": "wormholeProgram"
            }
          }
        },
        {
          "name": "message",
          "docs": [
            "Wormhole Message. The Wormhole Core Bridge program's post message",
            "instruction requires this account to be a mutable signer.",
            "",
            "This program uses a PDA per emitter. Messages are already bottle-necked",
            "by emitter sequence and the Wormhole Core Bridge program enforces that",
            "emitter must be identical for reused accounts. While this could be",
            "managed by the integrator, it seems more effective to have this Shim",
            "program manage these accounts."
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "account",
                "path": "emitter"
              }
            ]
          }
        },
        {
          "name": "emitter",
          "docs": [
            "Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge",
            "program's post message instruction requires this account to be a signer."
          ],
          "signer": true
        },
        {
          "name": "sequence",
          "docs": [
            "Emitter's sequence account. Wormhole Core Bridge program's post message",
            "instruction requires this account to be mutable."
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  83,
                  101,
                  113,
                  117,
                  101,
                  110,
                  99,
                  101
                ]
              },
              {
                "kind": "account",
                "path": "emitter"
              }
            ],
            "program": {
              "kind": "account",
              "path": "wormholeProgram"
            }
          }
        },
        {
          "name": "payer",
          "docs": [
            "Payer will pay the rent for the Wormhole Core Bridge emitter sequence",
            "and message on the first post message call. Subsequent calls will not",
            "require more lamports for rent."
          ],
          "writable": true,
          "signer": true
        },
        {
          "name": "feeCollector",
          "docs": [
            "Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post",
            "message instruction requires this account to be mutable."
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  102,
                  101,
                  101,
                  95,
                  99,
                  111,
                  108,
                  108,
                  101,
                  99,
                  116,
                  111,
                  114
                ]
              }
            ],
            "program": {
              "kind": "account",
              "path": "wormholeProgram"
            }
          }
        },
        {
          "name": "clock",
          "docs": [
            "Clock sysvar."
          ],
          "address": "SysvarC1ock11111111111111111111111111111111"
        },
        {
          "name": "systemProgram",
          "docs": [
            "System program."
          ],
          "address": "11111111111111111111111111111111"
        },
        {
          "name": "wormholeProgram",
          "docs": [
            "Wormhole Core Bridge program."
          ]
        },
        {
          "name": "eventAuthority",
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  95,
                  95,
                  101,
                  118,
                  101,
                  110,
                  116,
                  95,
                  97,
                  117,
                  116,
                  104,
                  111,
                  114,
                  105,
                  116,
                  121
                ]
              }
            ]
          }
        },
        {
          "name": "program"
        }
      ],
      "args": [
        {
          "name": "nonce",
          "type": "u32"
        },
        {
          "name": "consistencyLevel",
          "type": {
            "defined": {
              "name": "finality"
            }
          }
        },
        {
          "name": "payload",
          "type": "bytes"
        }
      ]
    }
  ],
  "events": [
    {
      "name": "messageEvent",
      "discriminator": [
        68,
        27,
        143,
        0,
        77,
        76,
        137,
        112
      ]
    }
  ],
  "types": [
    {
      "name": "finality",
      "type": {
        "kind": "enum",
        "variants": [
          {
            "name": "confirmed"
          },
          {
            "name": "finalized"
          }
        ]
      }
    },
    {
      "name": "messageEvent",
      "type": {
        "kind": "struct",
        "fields": [
          {
            "name": "emitter",
            "type": "pubkey"
          },
          {
            "name": "sequence",
            "type": "u64"
          },
          {
            "name": "submissionTime",
            "type": "u32"
          }
        ]
      }
    }
  ]
};

