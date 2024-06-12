import yargs from "yargs";
import { NETWORKS } from "../../consts";
import { chainToChain, getNetwork } from "../../utils";

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
      type: "string",
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);
  console.log(NETWORKS[network][chainToChain(argv.chain)].rpc);
};
