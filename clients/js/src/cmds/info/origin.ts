// import { tryUint8ArrayToNative } from "@certusone/wormhole-sdk/lib/esm/utils";
import yargs from "yargs";
import { getOriginalAsset_old } from "../../chains/generic";
import { CHAIN_ID_OR_NAME_CHOICES, RPC_OPTIONS } from "../../consts";
import { getNetwork } from "../../utils";
import { tryUint8ArrayToNative } from "../../array";
import { toChain } from "@wormhole-foundation/sdk-base";

export const command = "origin <chain> <address>";
export const desc = `Print the origin chain and address of the asset that corresponds to the given chain and address.`;
export const builder = (y: typeof yargs) =>
  y
    .positional("chain", {
      describe: "Chain that wrapped asset came from",
      choices: CHAIN_ID_OR_NAME_CHOICES,
      demandOption: true,
    } as const)
    .positional("address", {
      describe: "Address of wrapped asset on origin chain",
      type: "string",
      demandOption: true,
    })
    .option("network", {
      alias: "n",
      describe: "Network of target chain",
      choices: ["mainnet", "testnet", "devnet"],
      default: "mainnet",
      demandOption: false,
    } as const)
    .option("rpc", RPC_OPTIONS);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const consoleWarnTemp = console.warn;
  console.warn = () => {};

  const network = getNetwork(argv.network);
  const res = await getOriginalAsset_old(argv.chain, network, argv.address);
  console.log({
    ...res,
    assetAddress: tryUint8ArrayToNative(res.assetAddress, toChain(res.chainId)),
  });

  console.warn = consoleWarnTemp;
};
