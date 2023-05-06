import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { CONTRACTS } from "../../consts";
import { assertNetwork } from "../../utils";
import { impossible } from "../../vaa";

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
      describe: "Chain to query",
      choices: Object.keys(CHAINS) as (keyof typeof CHAINS)[],
      demandOption: true,
    } as const)
    .positional("module", {
      describe: "Module to query",
      choices: ["Core", "NFTBridge", "TokenBridge"],
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = argv.network.toUpperCase();
  assertNetwork(network);
  const chain = argv["chain"];
  assertChain(chain);
  const module = argv["module"];

  let addr: string | undefined;
  switch (module) {
    case "Core":
      addr = CONTRACTS[network][chain].core;
      break;
    case "NFTBridge":
      const addresses = CONTRACTS[network][chain];
      if (!("nft_bridge" in addresses)) {
        throw new Error(`NFTBridge not deployed on ${chain}`);
      }

      addr = addresses.nft_bridge;
      break;
    case "TokenBridge":
      addr = CONTRACTS[network][chain].token_bridge;
      break;
    default:
      impossible(module);
  }

  if (!addr) {
    throw new Error(`${module} not deployed on ${chain}`);
  }

  console.log(addr);
};
