import yargs from "yargs";
import { NETWORKS } from "../../consts";
import { assertChain, chains } from "@wormhole-foundation/sdk-base";
import { getNetwork } from "../../utils";

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
      choices: chains,
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  assertChain(argv.chain);
  const network = getNetwork(argv.network);
  console.log(NETWORKS[network][argv.chain].rpc);
};
