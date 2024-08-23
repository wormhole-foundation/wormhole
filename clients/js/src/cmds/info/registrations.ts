// The registration command queries the TokenBridge or NFTBridge for all bridges registered with it. By default,
// it prints out the results. Optionally, you can specify --verify to have it verify the registrations against what
// is defined in the consts.ts file in the SDK (to verify that all chains // are properly registered.)

import yargs from "yargs";
import { getEmitterAddress } from "../../emitter";
import {
  Network,
  chainToPlatform,
  chains,
  contracts,
  toChain,
} from "@wormhole-foundation/sdk-base";
import { chainToChain, getNetwork } from "../../utils";
import { Chain } from "@wormhole-foundation/sdk";

export const command = "registrations <network> <chain> <module>";
export const desc = "Print chain registrations";
export const builder = (y: typeof yargs) => {
  return y
    .positional("network", {
      describe: "network",
      choices: ["mainnet", "testnet", "devnet"],
      demandOption: true,
    } as const)
    .positional("chain", {
      describe:
        "Chain to query. To see a list of supported chains, run `worm chains`",
      type: "string",
      demandOption: true,
    } as const)
    .positional("module", {
      describe: "Module to query (TokenBridge or NFTBridge)",
      type: "string",
      choices: ["NFTBridge", "TokenBridge"],
      demandOption: true,
    })
    .option("verify", {
      alias: "v",
      describe: "Verify the results against the const file",
      type: "boolean",
      default: false,
      demandOption: false,
    });
};
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const chain = chainToChain(argv.chain);
  const network = getNetwork(argv.network);
  const module = argv.module;
  if (module !== "TokenBridge" && module !== "NFTBridge") {
    throw Error(`Module must be TokenBridge or NFTBridge`);
  }
  let results: object;
  if (chain === "Solana") {
    const solana = require("../../solana");
    results = await solana.queryRegistrationsSolana(network, module);
  } else if (chainToPlatform(chain) === "Evm") {
    const evm = require("../../evm");
    results = await evm.queryRegistrationsEvm(network, chain, module);
  } else if (chain === "Terra" || chain === "Terra2" || chain === "Xpla") {
    const terra = require("../../terra");
    results = await terra.queryRegistrationsTerra(network, chain, module);
  } else if (chain === "Injective") {
    const injective = require("../../injective");
    results = await injective.queryRegistrationsInjective(network, module);
  } else if (chain === "Sei") {
    const sei = require("../../chains/sei/registrations");
    results = await sei.queryRegistrationsSei(network, module);
  } else if (chain === "Sui") {
    const sui = require("../../chains/sui/registrations");
    results = await sui.queryRegistrationsSui(network, module);
  } else if (chain === "Aptos") {
    const aptos = require("../../aptos");
    results = await aptos.queryRegistrationsAptos(network, module);
  } else {
    throw Error(`Command not supported for chain ${chain}`);
  }
  if (argv["verify"]) {
    verifyRegistrations(network, chain, module, results);
  } else {
    console.log(results);
  }
};

// verifyRegistrations takes the results returned above and verifies them against the expected values in the consts file.
async function verifyRegistrations(
  network: Network,
  chain: Chain,
  module: "NFTBridge" | "TokenBridge",
  input: Object
) {
  let mismatchFound = false;

  // Put the input in a map so we can do lookups.
  let inputMap = new Map<string, string>();
  for (const [cname, reg] of Object.entries(input)) {
    inputMap.set(cname as string, reg as string);
  }

  // Loop over the chains and make sure everything is in our input, and the values match.
  const results: { [key: string]: string } = {};
  for (const chainStr of chains) {
    const thisChain = toChain(chainStr);
    if (thisChain === chain) {
      continue;
    }

    let expectedAddr: string | undefined;
    if (module === "TokenBridge") {
      expectedAddr = contracts.tokenBridge.get(network, thisChain);
    } else {
      expectedAddr = contracts.nftBridge.get(network, thisChain);
    }

    if (expectedAddr !== undefined) {
      expectedAddr = await getEmitterAddress(thisChain, expectedAddr);
      if (!expectedAddr.startsWith("0x")) {
        expectedAddr = "0x" + expectedAddr;
      }
    }

    let actualAddr = inputMap.get(thisChain as string);
    if (actualAddr !== undefined && !actualAddr.startsWith("0x")) {
      actualAddr = "0x" + actualAddr;
    }
    if (expectedAddr !== undefined) {
      if (
        actualAddr === undefined ||
        actualAddr ===
          "0x0000000000000000000000000000000000000000000000000000000000000000"
      ) {
        results[thisChain] = "Missing " + expectedAddr;
        mismatchFound = true;
      } else if (actualAddr !== expectedAddr) {
        results[thisChain] =
          "Expected " + expectedAddr + ", found " + actualAddr;
        mismatchFound = true;
      }
    } else if (
      actualAddr !== undefined &&
      actualAddr !==
        "0x0000000000000000000000000000000000000000000000000000000000000000"
    ) {
      results[thisChain] = "Expected null , found " + actualAddr;
      mismatchFound = true;
    }
  }

  if (mismatchFound) {
    console.log(`Mismatches found on  ${chain} ${network}!`);
    console.log(results);
  } else {
    console.log(`Verification of ${chain} ${network} succeeded!`);
  }
}
