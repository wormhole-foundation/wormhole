import { assertChain } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { getWrappedAssetAddress } from "../../chains/generic/getWrappedAssetAddress";
import { RPC_OPTIONS } from "../../consts";
import { chainToChain, getNetwork } from "../../utils";

export const command = "wrapped <origin-chain> <origin-address> <target-chain>";
export const desc =
  "Print the wrapped address on the target chain that corresponds with the specified origin chain and address.";
export const builder = (y: typeof yargs) =>
  y
    .positional("origin-chain", {
      describe: "Chain that wrapped asset came from",
      type: "string",
      demandOption: true,
    } as const)
    .positional("origin-address", {
      describe: "Address of wrapped asset on origin chain",
      type: "string",
      demandOption: true,
    })
    .positional("target-chain", {
      describe: "Chain to query for wrapped asset address",
      type: "string",
      demandOption: true,
    } as const)
    .option("network", {
      alias: "n",
      describe: "Network of target chain",
      choices: ["mainnet", "testnet", "devnet"],
      default: "mainnet",
      demandOption: false,
    } as const)
    .option("rpc", RPC_OPTIONS)
    .example(
      "worm info wrapped ethereum 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 solana",
      "A9mUU4qviSctJVPJdBJWkb28deg915LYJKrzQ19ji3FM"
    );
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const consoleWarnTemp = console.warn;
  console.warn = () => {};

  const originChain = chainToChain(argv["origin-chain"]);
  const originAddress = argv["origin-address"];
  const targetChain = chainToChain(argv["target-chain"]);
  const network = getNetwork(argv.network);

  assertChain(originChain);
  assertChain(targetChain);

  console.log(
    await getWrappedAssetAddress(
      targetChain,
      network,
      originChain,
      originAddress,
      argv.rpc
    )
  );

  console.warn = consoleWarnTemp;
};
