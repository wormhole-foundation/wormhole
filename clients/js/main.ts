#!/usr/bin/env node
import yargs from "yargs";

import { hideBin } from "yargs/helpers";

import { Bech32, fromBech32, toHex } from "@cosmjs/encoding";
import { isTerraChain, assertEVMChain, CONTRACTS, setDefaultWasm } from "@certusone/wormhole-sdk";
import { execute_solana } from "./solana";
import { execute_evm, getImplementation, hijack_evm, query_contract_evm, setStorageAt } from "./evm";
import { execute_terra } from "./terra";
import * as vaa from "./vaa";
import { impossible, Payload, serialiseVAA, VAA } from "./vaa";
import {
  assertChain,
  ChainName,
  CHAINS,
  toChainName,
  isEVMChain,
  toChainId,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { NETWORKS } from "./networks";
import base58 from "bs58";

setDefaultWasm("node");

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";

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
                emitterAddress: parseAddress(argv["chain"], argv["contract-address"]),
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
                address: parseCodeAddress(argv["chain"], argv["contract-address"]),
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
        buf = Buffer.from(String(argv.vaa), "hex")
        if (buf.length == 0) {
          throw Error("Couldn't parse VAA as hex")
        }
      } catch (e) {
        buf = Buffer.from(String(argv.vaa), "base64")
        if (buf.length == 0) {
          throw Error("Couldn't parse VAA as base64 or hex")
        }
      }
      const parsed_vaa = vaa.parse(buf);
      let parsed_vaa_with_digest = parsed_vaa;
      parsed_vaa_with_digest['digest'] = vaa.vaaDigest(parsed_vaa);
      console.log(parsed_vaa_with_digest);
    })
  .command("recover <digest> <signature>", "Recover an address from a signature", (yargs) => {
    return yargs
      .positional("digest", {
        describe: "digest",
        type: "string"
      })
      .positional("signature", {
        describe: "signature",
        type: "string"
      });
  }, async (argv) => {
    console.log(ethers.utils.recoverAddress(hex(argv["digest"]), hex(argv["signature"])))
  })
  .command("contract <network> <chain> <module>", "Print contract address", (yargs) => {
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
  }, async (argv) => {
    assertChain(argv["chain"])
    assertEVMChain(argv["chain"])
    const network = argv.network.toUpperCase();
    if (
      network !== "MAINNET" &&
      network !== "TESTNET" &&
      network !== "DEVNET"
    ) {
      throw Error(`Unknown network: ${network}`);
    }
    let module = argv["module"] as
      | "Core"
      | "NFTBridge"
      | "TokenBridge";
    switch (module) {
      case "Core":
        console.log(CONTRACTS[network][argv["chain"]]["core"])
        break;
      case "NFTBridge":
        console.log(CONTRACTS[network][argv["chain"]]["nft_bridge"])
        break;
      case "TokenBridge":
        console.log(CONTRACTS[network][argv["chain"]]["token_bridge"])
        break;
      default:
        impossible(module)
    }
  })
  .command("rpc <network> <chain>", "Print RPC address", (yargs) => {
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
  }, async (argv) => {
    assertChain(argv["chain"])
    assertEVMChain(argv["chain"])
    const network = argv.network.toUpperCase();
    if (
      network !== "MAINNET" &&
      network !== "TESTNET" &&
      network !== "DEVNET"
    ) {
      throw Error(`Unknown network: ${network}`);
    }
    console.log(NETWORKS[network][argv["chain"]].rpc)
  })
  ////////////////////////////////////////////////////////////////////////////////
  // Evm utilities
  .command("evm", "EVM utilites", (yargs) => {
    return yargs
      .option("rpc", {
        describe: "RPC endpoint",
        type: "string",
        required: false
      })
      .command("address-from-secret <secret>", "Compute a 20 byte eth address from a 32 byte private key", (yargs) => {
        return yargs
          .positional("secret", { type: "string", describe: "Secret key (32 bytes)" })
      }, (argv) => {
        console.log(ethers.utils.computeAddress(argv["secret"]))
      })
      .command("storage-update", "Update a storage slot on an EVM fork during testing (anvil or hardhat)", (yargs) => {
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
      }, async (argv) => {
        const result = await setStorageAt(argv["rpc"], evm_address(argv["contract-address"]), argv["storage-slot"], ["uint256"], [argv["value"]]);
        console.log(result);
      })
      .command("chains", "Return all EVM chains",
        async (_) => {
          console.log(Object.values(CHAINS).map(id => toChainName(id)).filter(name => isEVMChain(name)).join(" "))
        }
      )
      .command("info", "Query info about the on-chain state of the contract", (yargs) => {
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
      }, async (argv) => {
        assertChain(argv["chain"])
        assertEVMChain(argv["chain"])
        const network = argv.network.toUpperCase();
        if (
          network !== "MAINNET" &&
          network !== "TESTNET" &&
          network !== "DEVNET"
        ) {
          throw Error(`Unknown network: ${network}`);
        }
        let module = argv["module"] as
          | "Core"
          | "NFTBridge"
          | "TokenBridge";
        let rpc = argv["rpc"] ?? NETWORKS[network][argv["chain"]].rpc
        if (argv["implementation-only"]) {
          console.log(await getImplementation(network, argv["chain"], module, argv["contract-address"], rpc))
        } else {
          console.log(JSON.stringify(await query_contract_evm(network, argv["chain"], module, argv["contract-address"], rpc), null, 2))
        }
      })
      .command("hijack", "Override the guardian set of the core bridge contract during testing (anvil or hardhat)", (yargs) => {
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
            describe: "New guardian set index (if unspecified, default to overriding the current index)",
            type: "number"
          });
      }, async (argv) => {
        const guardian_addresses = argv["guardian-address"].split(",")
        let rpc = argv["rpc"] ?? NETWORKS.DEVNET.ethereum.rpc
        await hijack_evm(rpc, argv["core-contract-address"], guardian_addresses, argv["guardian-set-index"])
      })
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
          required: false
        })
    }, async (argv) => {
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
        await execute_evm(parsed_vaa.payload, buf, network, chain, argv["contract-address"], argv["rpc"]);
      } else if (isTerraChain(chain)) {
        await execute_terra(parsed_vaa.payload, buf, network, chain);
      } else if (chain === "solana") {
        await execute_solana(parsed_vaa, buf, network);
      } else if (chain === "algorand") {
        throw Error("Algorand is not supported yet");
      } else if (chain === "near") {
        throw Error("NEAR is not supported yet");
      } else if (chain === "injective") {
        throw Error("INJECTIVE is not supported yet");
      } else if (chain === "osmosis") {
        throw Error("OSMOSIS is not supported yet");
      } else if (chain === "sui") {
        throw Error("SUI is not supported yet");
      } else if (chain === "aptos") {
        throw Error("APTOS is not supported yet");
      } else {
        // If you get a type error here, hover over `chain`'s type and it tells you
        // which cases are not handled
        impossible(chain);
      }
    }
  ).argv;

function hex(x: string): string {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true })
}

function evm_address(x: string): string {
  return hex(x).substring(2).padStart(64, "0")
}

function parseAddress(chain: ChainName, address: string): string {
  if (chain === "unset") {
    throw Error("Chain unset")
  } else if (isEVMChain(chain)) {
    return "0x" + evm_address(address)
  } else if (isTerraChain(chain)) {
    return "0x" + toHex(fromBech32(address).data).padStart(64, "0")
  } else if (chain === "solana") {
    return "0x" + toHex(base58.decode(address)).padStart(64, "0")
  } else if (chain === "algorand") {
    // TODO: is there a better native format for algorand?
    return "0x" + evm_address(address)
  } else if (chain === "near") {
    return "0x" + evm_address(address)
  } else if (chain === "injective") {
    throw Error("INJECTIVE is not supported yet");
  } else if (chain === "osmosis") {
    throw Error("OSMOSIS is not supported yet");
  } else if (chain === "sui") {
    throw Error("SUI is not supported yet")
  } else if (chain === "aptos") {
    throw Error("APTOS is not supported yet")
  } else {
    impossible(chain)
  }
}

function parseCodeAddress(chain: ChainName, address: string): string {
  if (isTerraChain(chain)) {
    return "0x" + parseInt(address, 10).toString(16).padStart(64, "0")
  } else {
    return parseAddress(chain, address)
  }
}
