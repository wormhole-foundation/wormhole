#!/usr/bin/env node

// <sigh>
// when the native secp256k1 is missing, the eccrypto library decides TO PRINT A MESSAGE TO STDOUT:
// https://github.com/bitchan/eccrypto/blob/a4f4a5f85ef5aa1776dfa1b7801cad808264a19c/index.js#L23
//
// do you use a CLI tool that depends on that library and try to pipe the output
// of the tool into another? tough luck
//
// for lack of a better way to stop this, we patch the console.info function to
// drop that particular message...
// </sigh>
const info = console.info;
console.info = function(x: string) {
  if (x != "secp256k1 unavailable, reverting to browser version") {
    info(x);
  }
};

import yargs from "yargs";

import { hideBin } from "yargs/helpers";

import { fromBech32, toHex } from "@cosmjs/encoding";
import * as vaa from "./vaa";
import { impossible, Payload, serialiseVAA, VAA } from "./vaa";
import { ethers } from "ethers";
import { NETWORKS } from "./networks";
import base58 from "bs58";
import { isOutdated } from "./cmds/update";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { assertChain, assertEVMChain, ChainName, CHAINS, CONTRACTS as SDK_CONTRACTS, isCosmWasmChain, isEVMChain, isTerraChain, toChainId, toChainName } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

setDefaultWasm("node");

if (isOutdated()) {
  console.error(
    "\x1b[33m%s\x1b[0m",
    "WARNING: 'worm' is out of date. Run 'worm update' to update."
  );
}

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";

// TODO: remove this once the aptos SDK changes are merged in
const OVERRIDES = {
  MAINNET: {
    aptos: {
      token_bridge:
        "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
      core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
    },
  },
  TESTNET: {
    aptos: {
      token_bridge:
        "0x576410486a2da45eee6c949c995670112ddf2fbeedab20350d506328eefc9d4f",
      core: "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
    },
  },
  DEVNET: {
    aptos: {
      token_bridge:
        "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31",
      core: "0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017",
    },
  },
};

export const CONTRACTS = {
  MAINNET: { ...SDK_CONTRACTS.MAINNET, ...OVERRIDES.MAINNET },
  TESTNET: { ...SDK_CONTRACTS.TESTNET, ...OVERRIDES.TESTNET },
  DEVNET: { ...SDK_CONTRACTS.DEVNET, ...OVERRIDES.DEVNET },
};

function makeVAA(
  emitterChain: number,
  emitterAddress: string,
  signers: string[],
  p: Payload
): VAA<Payload> {
  let v: VAA<Payload> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 1,
    nonce: 1,
    emitterChain: emitterChain,
    emitterAddress: emitterAddress,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: p,
  };
  v.signatures = vaa.sign(signers, v);
  return v;
}

yargs(hideBin(process.argv))
  //TODO(csongor): refactor all commands into the directory structure.
  .commandDir("cmds")
  ////////////////////////////////////////////////////////////////////////////////
  // Generate
  .command(
    "generate",
    "generate VAAs (devnet and testnet only)",
    (yargs) => {
      return (
        yargs
          .option("guardian-secret", {
            alias: "g",
            required: true,
            describe: "Guardians' secret keys (CSV)",
            type: "string",
          })
          // Registration
          .command(
            "registration",
            "Generate registration VAA",
            (yargs) => {
              return yargs
                .option("chain", {
                  alias: "c",
                  describe: "Chain to register",
                  type: "string",
                  choices: Object.keys(CHAINS),
                  required: true,
                })
                .option("contract-address", {
                  alias: "a",
                  describe: "Contract to register",
                  type: "string",
                  required: true,
                })
                .option("module", {
                  alias: "m",
                  describe: "Module to upgrade",
                  type: "string",
                  choices: ["NFTBridge", "TokenBridge"],
                  required: true,
                });
            },
            (argv) => {
              let module = argv["module"] as "NFTBridge" | "TokenBridge";
              assertChain(argv["chain"]);
              let payload: vaa.PortalRegisterChain<typeof module> = {
                module,
                type: "RegisterChain",
                chain: 0,
                emitterChain: toChainId(argv["chain"]),
                emitterAddress: parseAddress(
                  argv["chain"],
                  argv["contract-address"]
                ),
              };
              let v = makeVAA(
                GOVERNANCE_CHAIN,
                GOVERNANCE_EMITTER,
                argv["guardian-secret"].split(","),
                payload
              );
              console.log(serialiseVAA(v));
            }
          )
          // Upgrade
          .command(
            "upgrade",
            "Generate contract upgrade VAA",
            (yargs) => {
              return yargs
                .option("chain", {
                  alias: "c",
                  describe: "Chain to upgrade",
                  type: "string",
                  choices: Object.keys(CHAINS),
                  required: true,
                })
                .option("contract-address", {
                  alias: "a",
                  describe: "Contract to upgrade to",
                  type: "string",
                  required: true,
                })
                .option("module", {
                  alias: "m",
                  describe: "Module to upgrade",
                  type: "string",
                  choices: ["Core", "NFTBridge", "TokenBridge"],
                  required: true,
                });
            },
            (argv) => {
              assertChain(argv["chain"]);
              let module = argv["module"] as
                | "Core"
                | "NFTBridge"
                | "TokenBridge";
              let payload: vaa.ContractUpgrade = {
                module,
                type: "ContractUpgrade",
                chain: toChainId(argv["chain"]),
                address: parseCodeAddress(
                  argv["chain"],
                  argv["contract-address"]
                ),
              };
              let v = makeVAA(
                GOVERNANCE_CHAIN,
                GOVERNANCE_EMITTER,
                argv["guardian-secret"].split(","),
                payload
              );
              console.log(serialiseVAA(v));
            }
          )
          .command(
            "attestation",
            "Generate a token attestation VAA",
            // TODO: putting 'any' here is a workaround for the following error:
            //
            //    Type instantiation is excessively deep and possibly infinite.
            //
            // The type of the yargs builder grows too big for typescript's
            // liking, and there's no way to increase the limit. So we
            // overapproximate with the 'any' type which reduces the typechecking stack.
            // This is not a great solution, and instead we should move toward
            // breaking up the commands into multiple modules in the 'cmds' folder.
            (yargs: any) => {
              return yargs
                .option("emitter-chain", {
                  alias: "e",
                  describe: "Emitter chain of the VAA",
                  type: "string",
                  choices: Object.keys(CHAINS),
                  required: true,
                })
                .option("emitter-address", {
                  alias: "f",
                  describe: "Emitter address of the VAA",
                  type: "string",
                  required: true,
                })
                .option("chain", {
                  alias: "c",
                  describe: "Token's chain",
                  type: "string",
                  choices: Object.keys(CHAINS),
                  required: true,
                })
                .option("token-address", {
                  alias: "a",
                  describe: "Token's address",
                  type: "string",
                  required: true,
                })
                .option("decimals", {
                  alias: "d",
                  describe: "Token's decimals",
                  type: "number",
                  required: true,
                })
                .option("symbol", {
                  alias: "s",
                  describe: "Token's symbol",
                  type: "string",
                  required: true,
                })
                .option("name", {
                  alias: "n",
                  describe: "Token's name",
                  type: "string",
                  required: true,
                });
            },
            (argv) => {
              let emitter_chain = argv["emitter-chain"] as string;
              assertChain(argv["chain"]);
              assertChain(emitter_chain);
              let payload: vaa.TokenBridgeAttestMeta = {
                module: "TokenBridge",
                type: "AttestMeta",
                chain: 0,
                // TODO: remove these casts (only here because of the workaround above)
                tokenAddress: parseAddress(
                  argv["chain"],
                  argv["token-address"] as string
                ),
                tokenChain: toChainId(argv["chain"]),
                decimals: argv["decimals"] as number,
                symbol: argv["symbol"] as string,
                name: argv["name"] as string,
              };
              let v = makeVAA(
                toChainId(emitter_chain),
                parseAddress(emitter_chain, argv["emitter-address"] as string),
                argv["guardian-secret"].split(","),
                payload
              );
              console.log(serialiseVAA(v));
            }
          )
      );
    },
    (_) => {
      yargs.showHelp();
    }
  )
  ////////////////////////////////////////////////////////////////////////////////
  // Misc
  .command(
    "parse <vaa>",
    "Parse a VAA (can be in either hex or base64 format)",
    (yargs) => {
      return yargs.positional("vaa", {
        describe: "vaa",
        type: "string",
      });
    },
    async (argv) => {
      let buf: Buffer;
      try {
        buf = Buffer.from(String(argv.vaa), "hex");
        if (buf.length == 0) {
          throw Error("Couldn't parse VAA as hex");
        }
      } catch (e) {
        buf = Buffer.from(String(argv.vaa), "base64");
        if (buf.length == 0) {
          throw Error("Couldn't parse VAA as base64 or hex");
        }
      }
      const parsed_vaa = vaa.parse(buf);
      let parsed_vaa_with_digest = parsed_vaa;
      parsed_vaa_with_digest["digest"] = vaa.vaaDigest(parsed_vaa);
      console.log(parsed_vaa_with_digest);
    }
  )
  .command(
    "recover <digest> <signature>",
    "Recover an address from a signature",
    (yargs) => {
      return yargs
        .positional("digest", {
          describe: "digest",
          type: "string",
        })
        .positional("signature", {
          describe: "signature",
          type: "string",
        });
    },
    async (argv) => {
      console.log(
        ethers.utils.recoverAddress(hex(argv["digest"]), hex(argv["signature"]))
      );
    }
  )
  .command(
    "contract <network> <chain> <module>",
    "Print contract address",
    (yargs) => {
      return yargs
        .positional("network", {
          describe: "network",
          type: "string",
          choices: ["mainnet", "testnet", "devnet"],
        })
        .positional("chain", {
          describe: "Chain to query",
          type: "string",
          choices: Object.keys(CHAINS),
        })
        .positional("module", {
          describe: "Module to query",
          type: "string",
          choices: ["Core", "NFTBridge", "TokenBridge"],
        })
        .option("emitter", {
          alias: "e",
          describe: "Print in emitter address format",
          type: "boolean",
          default: false,
          required: false,
        });
    },
    async (argv) => {
      assertChain(argv["chain"]);
      const network = argv.network.toUpperCase();
      if (
        network !== "MAINNET" &&
        network !== "TESTNET" &&
        network !== "DEVNET"
      ) {
        throw Error(`Unknown network: ${network}`);
      }
      let chain = argv["chain"];
      let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
      let addr = "";
      switch (module) {
        case "Core":
          addr = CONTRACTS[network][chain]["core"];
          break;
        case "NFTBridge":
          addr = CONTRACTS[network][chain]["nft_bridge"];
          break;
        case "TokenBridge":
          addr = CONTRACTS[network][chain]["token_bridge"];
          break;
        default:
          impossible(module);
      }
      if (argv["emitter"]) {
        const emitter = require("@certusone/wormhole-sdk/lib/cjs/bridge/getEmitterAddress")
        if (chain === "solana" || chain === "pythnet") {
          // TODO: Create an isSolanaChain()
          addr = await emitter.getEmitterAddressSolana(addr);
        } else if (emitter.isCosmWasmChain(chain)) {
          addr = await emitter.getEmitterAddressTerra(addr);
        } else if (chain === "algorand") {
          addr = emitter.getEmitterAddressAlgorand(BigInt(addr));
        } else if (chain === "near") {
          addr = emitter.getEmitterAddressNear(addr);
        } else {
          addr = emitter.getEmitterAddressEth(addr);
        }
      }
      console.log(addr);
    }
  )
  .command(
    "chain-id <chain>",
    "Print the wormhole chain ID integer associated with the specified chain name",
    (yargs) => {
      return yargs.positional("chain", {
        describe: "Chain to query",
        type: "string",
        choices: Object.keys(CHAINS),
      });
    },
    async (argv) => {
      assertChain(argv["chain"]);
      console.log(toChainId(argv["chain"]));
    }
  )
  .command(
    "rpc <network> <chain>",
    "Print RPC address",
    (yargs) => {
      return yargs
        .positional("network", {
          describe: "network",
          type: "string",
          choices: ["mainnet", "testnet", "devnet"],
        })
        .positional("chain", {
          describe: "Chain to query",
          type: "string",
          choices: Object.keys(CHAINS),
        });
    },
    async (argv) => {
      assertChain(argv["chain"]);
      const network = argv.network.toUpperCase();
      if (
        network !== "MAINNET" &&
        network !== "TESTNET" &&
        network !== "DEVNET"
      ) {
        throw Error(`Unknown network: ${network}`);
      }
      console.log(NETWORKS[network][argv["chain"]].rpc);
    }
  )
  ////////////////////////////////////////////////////////////////////////////////
  // Near utilities
  .command(
    "near",
    "NEAR utilites",
    (yargs) => {
      const near = require("./near")
      return (
        yargs
          .option("module", {
            alias: "m",
            describe: "Module to query",
            type: "string",
            choices: ["Core", "NFTBridge", "TokenBridge"],
            required: false,
          })
          .option("network", {
            alias: "n",
            describe: "network",
            type: "string",
            choices: ["mainnet", "testnet", "devnet"],
            required: true,
          })
          .option("account", {
            describe: "near deployment account",
            type: "string",
            required: true,
          })
          .option("attach", {
            describe: "attach some near",
            type: "string",
            required: false,
          })
          .option("target", {
            describe: "near account to upgrade",
            type: "string",
            required: false,
          })
          .option("mnemonic", {
            describe: "near private keys",
            type: "string",
            required: false,
          })
          .option("keys", {
            describe: "near private keys",
            type: "string",
            required: false,
          })
          .command(
            "contract-update <file>",
            "Submit a contract update using our specific APIs",
            (yargs) => {
              return yargs.positional("file", {
                type: "string",
                describe: "wasm",
              });
            },
            async (argv) => {
              await near.upgrade_near(argv);
            }
          )
          .command(
            "deploy <file>",
            "Submit a contract update using near APIs",
            (yargs) => {
              return yargs.positional("file", {
                type: "string",
                describe: "wasm",
              });
            },
            async (argv) => {
              await near.deploy_near(argv);
            }
          )
      );
    },
    (_) => {
      yargs.showHelp();
    }
  )

  ////////////////////////////////////////////////////////////////////////////////
  // Evm utilities
  .command(
    "evm",
    "EVM utilites",
    (yargs) => {
      const evm = require("./evm")
      return yargs
        .option("rpc", {
          describe: "RPC endpoint",
          type: "string",
          required: false,
        })
        .command(
          "address-from-secret <secret>",
          "Compute a 20 byte eth address from a 32 byte private key",
          (yargs) => {
            return yargs.positional("secret", {
              type: "string",
              describe: "Secret key (32 bytes)",
            });
          },
          (argv) => {
            console.log(ethers.utils.computeAddress(argv["secret"]));
          }
        )
        .command(
          "storage-update",
          "Update a storage slot on an EVM fork during testing (anvil or hardhat)",
          (yargs) => {
            return yargs
              .option("contract-address", {
                alias: "a",
                describe: "Contract address",
                type: "string",
                required: true,
              })
              .option("storage-slot", {
                alias: "k",
                describe: "Storage slot to modify",
                type: "string",
                required: true,
              })
              .option("value", {
                alias: "v",
                describe: "Value to write into the slot (32 bytes)",
                type: "string",
                required: true,
              });
          },
          async (argv) => {
            const result = await evm.setStorageAt(
              argv["rpc"],
              evm_address(argv["contract-address"]),
              argv["storage-slot"],
              ["uint256"],
              [argv["value"]]
            );
            console.log(result);
          }
        )
        .command("chains", "Return all EVM chains", async (_) => {
          console.log(
            Object.values(CHAINS)
              .map((id) => toChainName(id))
              .filter((name) => isEVMChain(name))
              .join(" ")
          );
        })
        .command(
          "info",
          "Query info about the on-chain state of the contract",
          (yargs) => {
            return yargs
              .option("chain", {
                alias: "c",
                describe: "Chain to query",
                type: "string",
                choices: Object.keys(CHAINS),
                required: true,
              })
              .option("module", {
                alias: "m",
                describe: "Module to query",
                type: "string",
                choices: ["Core", "NFTBridge", "TokenBridge"],
                required: true,
              })
              .option("network", {
                alias: "n",
                describe: "network",
                type: "string",
                choices: ["mainnet", "testnet", "devnet"],
                required: true,
              })
              .option("contract-address", {
                alias: "a",
                describe: "Contract to query (override config)",
                type: "string",
                required: false,
              })
              .option("implementation-only", {
                alias: "i",
                describe: "Only query implementation (faster)",
                type: "boolean",
                default: false,
                required: false,
              });
          },
          async (argv) => {
            assertChain(argv["chain"]);
            assertEVMChain(argv["chain"]);
            const network = argv.network.toUpperCase();
            if (
              network !== "MAINNET" &&
              network !== "TESTNET" &&
              network !== "DEVNET"
            ) {
              throw Error(`Unknown network: ${network}`);
            }
            let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
            let rpc = argv["rpc"] ?? NETWORKS[network][argv["chain"]].rpc;
            if (argv["implementation-only"]) {
              console.log(
                await evm.getImplementation(
                  network,
                  argv["chain"],
                  module,
                  argv["contract-address"],
                  rpc
                )
              );
            } else {
              console.log(
                JSON.stringify(
                  await evm.query_contract_evm(
                    network,
                    argv["chain"],
                    module,
                    argv["contract-address"],
                    rpc
                  ),
                  null,
                  2
                )
              );
            }
          }
        )
        .command(
          "hijack",
          "Override the guardian set of the core bridge contract during testing (anvil or hardhat)",
          (yargs) => {
            return yargs
              .option("core-contract-address", {
                alias: "a",
                describe: "Core contract address",
                type: "string",
                default: CONTRACTS.MAINNET.ethereum.core,
              })
              .option("guardian-address", {
                alias: "g",
                required: true,
                describe: "Guardians' public addresses (CSV)",
                type: "string",
              })
              .option("guardian-set-index", {
                alias: "i",
                required: false,
                describe:
                  "New guardian set index (if unspecified, default to overriding the current index)",
                type: "number",
              });
          },
          async (argv) => {
            const guardian_addresses = argv["guardian-address"].split(",");
            let rpc = argv["rpc"] ?? NETWORKS.DEVNET.ethereum.rpc;
            await evm.hijack_evm(
              rpc,
              argv["core-contract-address"],
              guardian_addresses,
              argv["guardian-set-index"]
            );
          }
        );
    },
    (_) => {
      yargs.showHelp();
    }
  )
  ////////////////////////////////////////////////////////////////////////////////
  // Submit
  .command(
    "submit <vaa>",
    "Execute a VAA",
    (yargs) => {
      // @ts-ignore
      return yargs
        .positional("vaa", {
          describe: "vaa",
          type: "string",
          required: true,
        })
        .option("chain", {
          alias: "c",
          describe: "chain name",
          type: "string",
          choices: Object.keys(CHAINS),
          required: false,
        })
        .option("network", {
          alias: "n",
          describe: "network",
          type: "string",
          choices: ["mainnet", "testnet", "devnet"],
          required: true,
        })
        .option("contract-address", {
          alias: "a",
          describe: "Contract to submit VAA to (override config)",
          type: "string",
          required: false,
        })
        .option("rpc", {
          describe: "RPC endpoint",
          type: "string",
          required: false,
        });
    },
    async (argv) => {
      const vaa_hex = String(argv.vaa);
      const buf = Buffer.from(vaa_hex, "hex");
      const parsed_vaa = vaa.parse(buf);

      vaa.assertKnownPayload(parsed_vaa);

      console.log(parsed_vaa.payload);

      const network = argv.network.toUpperCase();
      if (
        network !== "MAINNET" &&
        network !== "TESTNET" &&
        network !== "DEVNET"
      ) {
        throw Error(`Unknown network: ${network}`);
      }

      // We figure out the target chain to submit the VAA to.
      // The VAA might specify this itself (for example a contract upgrade VAA
      // or a token transfer VAA), in which case we just submit the VAA to
      // that target chain.
      //
      // If the VAA does not have a target (e.g. chain registration VAAs or
      // guardian set upgrade VAAs), we require the '--chain' argument to be
      // set on the command line.
      //
      // As a sanity check, in the event that the VAA does specify a target
      // and the '--chain' argument is also set, we issue an error if those
      // two don't agree instead of silently taking the VAA's target chain.

      // get VAA chain
      const vaa_chain_id = parsed_vaa.payload.chain;
      assertChain(vaa_chain_id);
      const vaa_chain = toChainName(vaa_chain_id);

      // get chain from command line arg
      const cli_chain = argv["chain"];

      let chain: ChainName;
      if (cli_chain !== undefined) {
        assertChain(cli_chain);
        if (vaa_chain !== "unset" && cli_chain !== vaa_chain) {
          throw Error(
            `Specified target chain (${cli_chain}) does not match VAA target chain (${vaa_chain})`
          );
        }
        chain = cli_chain;
      } else {
        chain = vaa_chain;
      }

      if (chain === "unset") {
        throw Error(
          "This VAA does not specify the target chain, please provide it by hand using the '--chain' flag."
        );
      } else if (isEVMChain(chain)) {
        const evm = require("./evm")
        await evm.execute_evm(
          parsed_vaa.payload,
          buf,
          network,
          chain,
          argv["contract-address"],
          argv["rpc"]
        );
      } else if (isTerraChain(chain)) {
        const terra = require("./terra")
        await terra.execute_terra(parsed_vaa.payload, buf, network, chain);
      } else if (chain === "solana" || chain === "pythnet") {
        const solana = require("./solana")
        await solana.execute_solana(parsed_vaa, buf, network, chain);
      } else if (chain === "algorand") {
        const algorand = require("./algorand")
        await algorand.execute_algorand(
          parsed_vaa.payload,
          Buffer.from(vaa_hex, "hex"),
          network
        );
      } else if (chain === "near") {
        const near = require("./near")
        await near.execute_near(parsed_vaa.payload, vaa_hex, network);
      } else if (chain === "injective") {
        const injective = require("./injective")
        await injective.execute_injective(parsed_vaa.payload, buf, network);
      } else if (chain === "xpla") {
        const xpla = require("./xpla")
        await xpla.execute_xpla(parsed_vaa.payload, buf, network);
      } else if (chain === "osmosis") {
        throw Error("OSMOSIS is not supported yet");
      } else if (chain === "sui") {
        throw Error("SUI is not supported yet");
      } else if (chain === "aptos") {
        const aptos = require("./aptos")
        await aptos.execute_aptos(
          parsed_vaa.payload,
          buf,
          network,
          argv["contract-address"],
          argv["rpc"]
        );
      } else if (chain == "wormholechain" || (chain+"") === "wormchain") {
        // TODO: update this condition after ChainName is updated to remove "wormholechain"
        throw Error("Wormchain is not supported yet");
      } else {
        // If you get a type error here, hover over `chain`'s type and it tells you
        // which cases are not handled
        impossible(chain);
      }
    }
  )
  .strict()
  .demandCommand().argv;

function hex(x: string): string {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true });
}

function evm_address(x: string): string {
  return hex(x).substring(2).padStart(64, "0");
}

function parseAddress(chain: ChainName, address: string): string {
  if (chain === "unset") {
    throw Error("Chain unset");
  } else if (isEVMChain(chain)) {
    return "0x" + evm_address(address);
  } else if (isCosmWasmChain(chain)) {
    return "0x" + toHex(fromBech32(address).data).padStart(64, "0");
  } else if (chain === "solana" || chain === "pythnet") {
    return "0x" + toHex(base58.decode(address)).padStart(64, "0");
  } else if (chain === "algorand") {
    // TODO: is there a better native format for algorand?
    return "0x" + evm_address(address);
  } else if (chain === "near") {
    return "0x" + hex(address).substring(2).padStart(64, "0");
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    throw Error("SUI is not supported yet");
  } else if (chain === "aptos") {
    // TODO: is there a better native format for aptos?
    return "0x" + evm_address(address);
  } else if (chain === "wormholechain" || (chain + "") == "wormchain") {
    // TODO: update this condition after ChainName is updated to remove "wormholechain"
    const sdk = require("@certusone/wormhole-sdk/lib/cjs/utils/array")
    return "0x" + sdk.tryNativeToHexString(address, chain);
  } else {
    impossible(chain);
  }
}

function parseCodeAddress(chain: ChainName, address: string): string {
  if (isTerraChain(chain)) {
    return "0x" + parseInt(address, 10).toString(16).padStart(64, "0");
  } else {
    return parseAddress(chain, address);
  }
}
