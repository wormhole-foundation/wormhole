import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { NETWORKS } from "../networks";
import { assertNetwork } from "../utils";

export const command = "rpc <network> <chain>";
export const desc = "Print RPC address";
export const builder = (y: typeof yargs) =>
  y
    .positional("network", {
      describe: "network",
      choices: ["mainnet", "testnet", "devnet"],
      demandOption: true,
    } as const)
    .positional("chain", {
      describe: "Chain to query",
      choices: Object.keys(CHAINS),
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  assertChain(argv["chain"]);
  const network = argv.network.toUpperCase();
  assertNetwork(network);
  console.log(NETWORKS[network][argv["chain"]].rpc);
};
