/**
 * Program IDL in camelCase format in order to be used in JS/TS.
 *
 * Note that this is only a type helper and is not the actual IDL. The original
 * IDL can be found at `target/idl/wormhole_post_message_shim.json`.
 */
export type WormholePostMessageShim = {
  address: "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX";
  metadata: {
    name: "wormholePostMessageShim";
    version: "0.1.0";
    spec: "0.1.0";
    description: "Created with Anchor";
  };
  instructions: [
    {
      name: "postMessage";
      docs: [
        "This instruction is intended to be a significantly cheaper alternative to `post_message` on the core bridge.",
        "It achieves this by reusing the message account, per emitter, via `post_message_unreliable` and",
        "emitting a CPI event for the guardian to observe containing the information previously only found",
        "in the resulting message account. Since this passes through the emitter and calls `post_message_unreliable`",
        "on the core bridge, it can be used (or not used) without disruption.",
        "",
        "NOTE: In the initial message publication for a new emitter, this will require one additional CPI call depth",
        "when compared to using the core bridge directly. If that is an issue, simply emit an empty message on initialization",
        "(or migration) in order to instantiate the account. This will result in a VAA from your emitter, so be careful to",
        "avoid any issues that may result in.",
        "",
        "Direct case",
        "shim `PostMessage` -> core `0x8`",
        "-> shim `MesssageEvent`",
        "",
        "Integration case",
        "Integrator Program -> shim `PostMessage` -> core `0x8`",
        "-> shim `MesssageEvent`"
      ];
      discriminator: [214, 50, 100, 209, 38, 34, 7, 76];
      accounts: [
        {
          name: "bridge";
          writable: true;
          address: "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP";
        },
        {
          name: "message";
          docs: [
            "This program uses a PDA per emitter, since these are already bottle-necked by sequence and",
            "the bridge enforces that emitter must be identical for reused accounts.",
            "While this could be managed by the integrator, it seems more effective to have the shim manage these accounts.",
            "Bonus, this also allows Anchor to automatically handle deriving the address."
          ];
          writable: true;
          pda: {
            seeds: [
              {
                kind: "account";
                path: "emitter";
              }
            ];
          };
        },
        {
          name: "emitter";
          signer: true;
        },
        {
          name: "sequence";
          docs: [
            "Explicitly do not re-derive this account. The core bridge verifies the derivation anyway and",
            "as of Anchor 0.30.1, auto-derivation for other programs' accounts via IDL doesn't work."
          ];
          writable: true;
        },
        {
          name: "payer";
          docs: ["Payer will pay Wormhole fee to post a message."];
          writable: true;
          signer: true;
        },
        {
          name: "feeCollector";
          writable: true;
          address: "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs";
        },
        {
          name: "clock";
          docs: ["Clock sysvar."];
          address: "SysvarC1ock11111111111111111111111111111111";
        },
        {
          name: "systemProgram";
          docs: ["System program."];
          address: "11111111111111111111111111111111";
        },
        {
          name: "wormholeProgram";
          address: "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
        },
        {
          name: "eventAuthority";
          pda: {
            seeds: [
              {
                kind: "const";
                value: [
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
                ];
              }
            ];
          };
        },
        {
          name: "program";
        }
      ];
      args: [
        {
          name: "nonce";
          type: "u32";
        },
        {
          name: "consistencyLevel";
          type: {
            defined: {
              name: "finality";
            };
          };
        },
        {
          name: "payload";
          type: "bytes";
        }
      ];
    }
  ];
  events: [
    {
      name: "messageEvent";
      discriminator: [68, 27, 143, 0, 77, 76, 137, 112];
    }
  ];
  types: [
    {
      name: "finality";
      type: {
        kind: "enum";
        variants: [
          {
            name: "confirmed";
          },
          {
            name: "finalized";
          }
        ];
      };
    },
    {
      name: "messageEvent";
      type: {
        kind: "struct";
        fields: [
          {
            name: "emitter";
            type: "pubkey";
          },
          {
            name: "sequence";
            type: "u64";
          },
          {
            name: "submissionTime";
            type: "u32";
          }
        ];
      };
    }
  ];
};
