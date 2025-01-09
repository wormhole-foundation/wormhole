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
    "version": "0.1.0",
    "spec": "0.1.0",
    "description": "Created with Anchor"
  },
  "instructions": [
    {
      "name": "postMessage",
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
          "writable": true,
          "address": "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
        },
        {
          "name": "message",
          "docs": [
            "This program uses a PDA per emitter, since these are already bottle-necked by sequence and",
            "the bridge enforces that emitter must be identical for reused accounts.",
            "While this could be managed by the integrator, it seems more effective to have the shim manage these accounts.",
            "Bonus, this also allows Anchor to automatically handle deriving the address."
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
          "signer": true
        },
        {
          "name": "sequence",
          "docs": [
            "Explicitly do not re-derive this account. The core bridge verifies the derivation anyway and",
            "as of Anchor 0.30.1, auto-derivation for other programs' accounts via IDL doesn't work."
          ],
          "writable": true
        },
        {
          "name": "payer",
          "docs": [
            "Payer will pay Wormhole fee to post a message."
          ],
          "writable": true,
          "signer": true
        },
        {
          "name": "feeCollector",
          "writable": true,
          "address": "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
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
          "name": "rent",
          "docs": [
            "Rent sysvar."
          ],
          "address": "SysvarRent111111111111111111111111111111111"
        },
        {
          "name": "wormholeProgram",
          "address": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
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
