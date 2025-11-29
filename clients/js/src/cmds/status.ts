import yargs from "yargs";
import { ethers } from "ethers";
import { NETWORKS } from "../consts";
import { castChainIdToOldSdk, chainToChain, getNetwork } from "../utils";
import {
  Chain,
  assertChain,
  chainToChainId,
  contracts,
} from "@wormhole-foundation/sdk-base";
import { ChainName, relayer, toChainName } from "@certusone/wormhole-sdk";

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
  const chain = chainToChain(argv.chain);
  assertChain(chain);

  const addr = contracts.relayer.get(network, chain);
  if (!addr) {
    throw new Error(`Wormhole Relayer not deployed on ${chain} in ${network}`);
  }

  const sourceRPC = NETWORKS[network][chain].rpc;
  const sourceChainProvider = new ethers.providers.JsonRpcProvider(sourceRPC);
  const targetChainProviders = new Map<ChainName, ethers.providers.Provider>();
  for (const key in NETWORKS[network]) {
    targetChainProviders.set(
      toChainName(castChainIdToOldSdk(chainToChainId(key as Chain))),
      new ethers.providers.JsonRpcProvider(NETWORKS[network][key as Chain].rpc)
    );
  }

  // TODO: Convert this over to sdkv2
  const v1ChainName = toChainName(castChainIdToOldSdk(chainToChainId(chain)));
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
