// The registration command queries the TokenBridge or NFTBridge for all bridges registered with it. By default,
// it prints out the results. Optionally, you can specify --verify to have it verify the registrations against what
// is defined in the consts.ts file in the SDK (to verify that all chains // are properly registered.)

import yargs from "yargs";
import {
  ChainName,
  CHAINS,
  CONTRACTS,
  assertChain,
  isEVMChain,
  isTerraChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { getEmitterAddress } from "../emitter";

export const command = "registrations <network> <chain> <module>";
export const desc = "Print chain registrations";
export const builder = (y: typeof yargs) => {
  return y
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
      describe: "Module to query (TokenBridge or NFTBridge)",
      type: "string",
      choices: ["NFTBridge", "TokenBridge"],
      required: true,
    })
    .option("verify", {
      alias: "v",
      describe: "Verify the results against the const file",
      type: "boolean",
      default: false,
      required: false,
    });
};
export const handler = async (argv) => {
  assertChain(argv["chain"]);
  const chain = argv.chain;
  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }
  const module = argv.module;
  if (module !== "TokenBridge" && module !== "NFTBridge") {
    throw Error(`Module must be TokenBridge or NFTBridge`);
  }
  let results: object;
  if (chain === "solana") {
    const solana = require("../solana");
    results = await solana.query_registrations_solana(network, module);
  } else if (isEVMChain(chain)) {
    const evm = require("../evm");
    results = await evm.query_registrations_evm(network, chain, module);
  } else if (isTerraChain(chain) || chain === "xpla") {
    const terra = require("../terra");
    results = await terra.query_registrations_terra(network, chain, module);
  } else if (chain === "injective") {
    const injective = require("../injective");
    await injective.query_registrations_injective(network, module);
  } else if (chain === "sei") {
    const sei = require("../sei");
    results = await sei.query_registrations_sei(network, module);
  } else if (chain === "sui") {
    const sui = require("../sui/registrations");
    results = await sui.query_registrations_sui(network, module);
  } else if (chain === "aptos") {
    const aptos = require("../aptos");
    results = await aptos.query_registrations_aptos(network, module);
  } else {
    throw Error(`Command not supported for chain ${chain}`);
  }
  if (argv["verify"]) {
    verify_registrations(network, chain as string, module, results);
  } else {
    console.log(results);
  }
};

// verify_registrations takes the results returned above and verifies them against the expected values in the consts file.
async function verify_registrations(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: string,
  module: "Core" | "NFTBridge" | "TokenBridge",
  input: Object
) {
  console.log(`Verifying results against consts file...`);
  let mismatchFound = false;

  // Put the input in a map so we can do lookups.
  let inputMap = new Map<string, string>();
  for (const [c_name, reg] of Object.entries(input)) {
    inputMap.set(c_name as string, reg as string);
  }

  // Loop over the consts and make sure everything is in our input, and the values match.
  let results = {};
  for (const c of Object.entries(CONTRACTS[network])) {
    if (c[0] === "unset" || c[0] === chain) {
      continue;
    }
    let expectedAddr: string;
    if (module === "TokenBridge") {
      expectedAddr = c[1].token_bridge;
    } else {
      expectedAddr = c[1].nft_bridge;
    }

    if (expectedAddr !== undefined) {
      expectedAddr =
        "0x" + (await getEmitterAddress(c[0] as ChainName, expectedAddr));
    }

    let actualAddr = inputMap.get(c[0] as string);
    if (expectedAddr !== undefined) {
      if (
        actualAddr === undefined ||
        actualAddr ===
          "0x0000000000000000000000000000000000000000000000000000000000000000"
      ) {
        results[c[0]] = "Missing " + expectedAddr;
        mismatchFound = true;
      } else if (actualAddr !== expectedAddr) {
        results[c[0]] = "Expected " + expectedAddr + ", found " + actualAddr;
        mismatchFound = true;
      }
    } else if (
      actualAddr !== undefined &&
      actualAddr !==
        "0x0000000000000000000000000000000000000000000000000000000000000000"
    ) {
      results[c[0]] = "Expected null , found " + actualAddr;
      mismatchFound = true;
    }
  }

  if (mismatchFound) {
    console.log("Mismatches found!");
    console.log(results);
  } else {
    console.log("Verification succeeded!");
  }
}
