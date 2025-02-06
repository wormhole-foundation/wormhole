/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/wormhole_integrator_example.json`.
 */
export type WormholeIntegratorExample = {
  "address": "AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ",
  "metadata": {
    "name": "wormholeIntegratorExample",
    "version": "0.1.0",
    "spec": "0.1.0",
    "description": "Created with Anchor"
  },
  "instructions": [
    {
      "name": "consumeVaa",
      "docs": [
        "This example instruction takes the guardian signatures account previously posted to the verify vaa shim",
        "with `post_signatures` and the corresponding guardian set from the core bridge and verifies a",
        "provided VAA body against it. `close_signatures` on the shim should be called immediately",
        "afterwards in order to reclaim the rent lamports taken by `post_signatures`."
      ],
      "discriminator": [
        224,
        143,
        180,
        192,
        139,
        120,
        177,
        63
      ],
      "accounts": [
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
          "name": "wormholeVerifyVaaShim",
          "address": "EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at"
        }
      ],
      "args": [
        {
          "name": "guardianSetBump",
          "type": "u8"
        },
        {
          "name": "vaaBody",
          "type": "bytes"
        }
      ]
    },
    {
      "name": "initialize",
      "docs": [
        "This example instruction posts an empty message during initialize in order to have one less",
        "CPI depth on subsequent posts.",
        "",
        "NOTE: this example does not replay protect the call. Typically, this may be done with an",
        "`init` constraint on an account that was used to store config that is also set up during initialization."
      ],
      "discriminator": [
        175,
        175,
        109,
        31,
        13,
        152,
        155,
        237
      ],
      "accounts": [
        {
          "name": "payer",
          "writable": true,
          "signer": true
        },
        {
          "name": "deployer",
          "signer": true
        },
        {
          "name": "programData",
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  137,
                  75,
                  208,
                  247,
                  32,
                  141,
                  13,
                  249,
                  158,
                  80,
                  9,
                  219,
                  168,
                  206,
                  79,
                  113,
                  223,
                  183,
                  165,
                  86,
                  124,
                  110,
                  87,
                  91,
                  73,
                  217,
                  240,
                  163,
                  59,
                  193,
                  96,
                  119
                ]
              }
            ]
          }
        },
        {
          "name": "wormholePostMessageShim",
          "address": "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX"
        },
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
            ],
            "program": {
              "kind": "const",
              "value": [
                206,
                93,
                34,
                116,
                131,
                143,
                202,
                41,
                198,
                209,
                143,
                152,
                10,
                211,
                213,
                245,
                235,
                78,
                129,
                210,
                121,
                29,
                243,
                98,
                128,
                136,
                144,
                147,
                38,
                68,
                208,
                24
              ]
            }
          }
        },
        {
          "name": "emitter",
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  101,
                  109,
                  105,
                  116,
                  116,
                  101,
                  114
                ]
              }
            ]
          }
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
          "name": "wormholeProgram",
          "address": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
        },
        {
          "name": "wormholePostMessageShimEa"
        }
      ],
      "args": []
    },
    {
      "name": "postMessage",
      "docs": [
        "This example instruction posts a message via the post message shim."
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
          "name": "payer",
          "writable": true,
          "signer": true
        },
        {
          "name": "wormholePostMessageShim",
          "address": "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX"
        },
        {
          "name": "bridge",
          "docs": [
            "Address constraint added for IDL generation / convenience, it will be enforced by the core bridge."
          ],
          "writable": true,
          "address": "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
        },
        {
          "name": "message",
          "docs": [
            "Seeds constraint added for IDL generation / convenience, it will be enforced by the shim."
          ],
          "writable": true,
          "pda": {
            "seeds": [
              {
                "kind": "account",
                "path": "emitter"
              }
            ],
            "program": {
              "kind": "const",
              "value": [
                206,
                93,
                34,
                116,
                131,
                143,
                202,
                41,
                198,
                209,
                143,
                152,
                10,
                211,
                213,
                245,
                235,
                78,
                129,
                210,
                121,
                29,
                243,
                98,
                128,
                136,
                144,
                147,
                38,
                68,
                208,
                24
              ]
            }
          }
        },
        {
          "name": "emitter",
          "docs": [
            "Seeds constraint added for IDL generation / convenience, it will be enforced to match the signer used in the CPI call."
          ],
          "pda": {
            "seeds": [
              {
                "kind": "const",
                "value": [
                  101,
                  109,
                  105,
                  116,
                  116,
                  101,
                  114
                ]
              }
            ]
          }
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
          "name": "feeCollector",
          "docs": [
            "Address constraint added for IDL generation / convenience, it will be enforced by the core bridge."
          ],
          "writable": true,
          "address": "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
        },
        {
          "name": "clock",
          "docs": [
            "Clock sysvar.",
            "Type added for IDL generation / convenience, it will be enforced by the core bridge."
          ],
          "address": "SysvarC1ock11111111111111111111111111111111"
        },
        {
          "name": "systemProgram",
          "docs": [
            "System program.",
            "Type for IDL generation / convenience, it will be enforced by the core bridge."
          ],
          "address": "11111111111111111111111111111111"
        },
        {
          "name": "wormholeProgram",
          "docs": [
            "Address constraint added for IDL generation / convenience, it will be enforced by the shim."
          ],
          "address": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
        },
        {
          "name": "wormholePostMessageShimEa",
          "docs": [
            "TODO: An address constraint could be included if this address was published to wormhole_solana_consts",
            "Address will be enforced by the shim."
          ]
        }
      ],
      "args": []
    }
  ]
};

