import yargs from "yargs";
import { chainToCliChain, getChainRpc, getNetwork } from "../../utils";

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
      describe:
        "Chain to query. To see a list of supported chains, run `worm chains`",
      type: "string",
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);
  const chain = chainToCliChain(argv.chain);
  console.log(getChainRpc(network, chain));
};
