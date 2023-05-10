import { tryUint8ArrayToNative } from "@certusone/wormhole-sdk/lib/esm/utils";
import yargs from "yargs";
import { getOriginalAsset } from "../../chains/generic";
import { CHAIN_ID_OR_NAME_CHOICES, RPC_OPTIONS } from "../../consts";
import { assertNetwork } from "../../utils";

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

  const network = argv.network.toUpperCase();
  assertNetwork(network);
  const res = await getOriginalAsset(argv.chain, network, argv.address);
  console.log({
    ...res,
    assetAddress: tryUint8ArrayToNative(res.assetAddress, res.chainId),
  });

  console.warn = consoleWarnTemp;
};
