import yargs from "yargs";
import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { NETWORKS } from "../networks";

exports.command = "rpc <network> <chain>";
exports.desc = "Print RPC address";
exports.builder = (y: typeof yargs) => {
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
    });
};
exports.handler = async (argv) => {
  assertChain(argv["chain"]);
  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }
  console.log(NETWORKS[network][argv["chain"]].rpc);
};
