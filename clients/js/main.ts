#!/usr/bin/env node
import yargs from "yargs";

import { hideBin } from "yargs/helpers";

import { isTerraChain, setDefaultWasm } from "@certusone/wormhole-sdk";
import { execute_governance_solana } from "./solana";
import { execute_governance_evm } from "./evm";
import { execute_governance_terra } from "./terra";
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
            describe: "Guardians' secret keys",
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
                emitterAddress: Buffer.from(
                  argv["contract-address"].padStart(64, "0"),
                  "hex"
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
              let payload: Payload = {
                module,
                type: "ContractUpgrade",
                chain: toChainId(argv["chain"]),
                address: Buffer.from(
                  argv["contract-address"].padStart(64, "0"),
                  "hex"
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
      );
    },
    (_) => {
      yargs.showHelp();
    }
  )
  ////////////////////////////////////////////////////////////////////////////////
  // Parse
  .command(
    "parse <vaa>",
    "Parse a VAA",
    (yargs) => {
      return yargs.positional("vaa", {
        describe: "vaa",
        type: "string",
      });
    },
    async (argv) => {
      const buf = Buffer.from(String(argv.vaa), "hex");
      const parsed_vaa = vaa.parse(buf);
      console.log(parsed_vaa);
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
        });
    },
    async (argv) => {
      const vaa_hex = String(argv.vaa);
      const buf = Buffer.from(vaa_hex, "hex");
      const parsed_vaa = vaa.parse(buf);

      if (!vaa.hasPayload(parsed_vaa)) {
        throw Error("Couldn't parse VAA payload");
      }

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
        await execute_governance_evm(parsed_vaa.payload, buf, network, chain);
      } else if (isTerraChain(chain)) {
        await execute_governance_terra(parsed_vaa.payload, buf, network);
      } else if (chain === "solana") {
        await execute_governance_solana(parsed_vaa, buf, network);
      } else if (chain === "algorand") {
        throw Error("Algorand is not supported yet");
      } else if (chain === "near") {
        throw Error("NEAR is not supported yet");
      } else {
        // If you get a type error here, hover over `chain`'s type and it tells you
        // which cases are not handled
        impossible(chain);
      }
    }
  ).argv;
