import yargs from "yargs";
import { ethers } from "ethers";
import { NETWORKS } from "../consts";
import {
  assertLiveChain,
  chainToCliChain,
  cliChainToChainId,
  getChainRpc,
  getNetwork,
  getRelayerContract,
} from "../utils";
import { Chain, chainToChainId } from "@wormhole-foundation/sdk-base";
import {
  CHAIN_ID_TO_NAME,
  ChainName,
  relayer,
  toChainName,
} from "@certusone/wormhole-sdk";

export const command = "status <network> <chain> <tx>";
export const desc =
  "Prints information about the automatic delivery initiated on the specified network, chain, and tx";
export const builder = (y: typeof yargs) =>
  y
    .positional("network", {
      describe: "Network",
      choices: ["mainnet", "testnet", "devnet"],
      demandOption: true,
    } as const)
    .positional("chain", {
      describe:
        "Source chain. To see a list of supported chains, run `worm chains`",
      type: "string",
      demandOption: true,
    } as const)
    .positional("tx", {
      describe: "Source transaction hash",
      type: "string",
      demandOption: true,
    } as const);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);
  const chain = chainToCliChain(argv.chain);
  assertLiveChain(chain, "relayer status cannot be queried for it");

  const addr = getRelayerContract(network, chain);
  if (!addr) {
    throw new Error(`Wormhole Relayer not deployed on ${chain} in ${network}`);
  }

  const sourceRPC = getChainRpc(network, chain);
  const sourceChainProvider = new ethers.providers.JsonRpcProvider(sourceRPC);
  const targetChainProviders = new Map<ChainName, ethers.providers.Provider>();
  for (const key in NETWORKS[network]) {
    const chainId = chainToChainId(key as Chain);
    // chains added after the legacy SDK was frozen can't be relayer targets
    if (!(chainId in CHAIN_ID_TO_NAME)) {
      continue;
    }
    targetChainProviders.set(
      toChainName(chainId as Parameters<typeof toChainName>[0]),
      new ethers.providers.JsonRpcProvider(NETWORKS[network][key as Chain].rpc)
    );
  }

  const sourceChainId = cliChainToChainId(chain);
  if (!(sourceChainId in CHAIN_ID_TO_NAME)) {
    throw new Error(`${chain} is not supported by the legacy relayer SDK`);
  }
  const v1ChainName = toChainName(
    sourceChainId as Parameters<typeof toChainName>[0]
  );
  const info = await relayer.getWormholeRelayerInfo(v1ChainName, argv.tx, {
    environment:
      network === "Devnet"
        ? "DEVNET"
        : network === "Testnet"
        ? "TESTNET"
        : "MAINNET",
    sourceChainProvider,
    targetChainProviders,
  });

  console.log(relayer.stringifyWormholeRelayerInfo(info));
};
