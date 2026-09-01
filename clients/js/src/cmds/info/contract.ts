import yargs from "yargs";
import { impossible } from "../../vaa";
import {
  chainToCliChain,
  getCoreContract,
  getNetwork,
  getNftBridgeContract,
  getRelayerContract,
  getTokenBridgeContract,
} from "../../utils";

export const command = "contract <network> <chain> <module>";
export const desc = "Print contract address";
export const builder = (y: typeof yargs) =>
  y
    .positional("network", {
      describe: "Network",
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
      describe: "Module to query",
      choices: ["Core", "NFTBridge", "TokenBridge", "WormholeRelayer"],
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);
  const chain = chainToCliChain(argv.chain);
  const module = argv["module"];

  let addr: string | undefined;
  switch (module) {
    case "Core":
      addr = getCoreContract(network, chain);
      break;
    case "NFTBridge":
      addr = getNftBridgeContract(network, chain);
      break;
    case "TokenBridge":
      addr = getTokenBridgeContract(network, chain);
      break;
    case "WormholeRelayer":
      addr = getRelayerContract(network, chain);
      break;
    default:
      impossible(module);
  }

  if (!addr) {
    throw new Error(`${module} not deployed on ${chain}`);
  }

  console.log(addr);
};
