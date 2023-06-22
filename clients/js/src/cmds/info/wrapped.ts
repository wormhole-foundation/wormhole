import { assertChain } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { getWrappedAssetAddress } from "../../chains/generic/getWrappedAssetAddress";
import { CHAIN_ID_OR_NAME_CHOICES, RPC_OPTIONS } from "../../consts";
import { assertNetwork } from "../../utils";

export const command = "wrapped <origin-chain> <origin-address> <target-chain>";
export const desc =
  "Print the wrapped address on the target chain that corresponds with the specified origin chain and address.";
export const builder = (y: typeof yargs) =>
  y
    .positional("origin-chain", {
      describe: "Chain that wrapped asset came from",
      choices: CHAIN_ID_OR_NAME_CHOICES,
      demandOption: true,
    } as const)
    .positional("origin-address", {
      describe: "Address of wrapped asset on origin chain",
      type: "string",
      demandOption: true,
    })
    .positional("target-chain", {
      describe: "Chain to query for wrapped asset address",
      choices: CHAIN_ID_OR_NAME_CHOICES,
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

  const originChain = argv["origin-chain"];
  const originAddress = argv["origin-address"];
  const targetChain = argv["target-chain"];
  const network = argv.network.toUpperCase();

  assertChain(originChain);
  assertChain(targetChain);
  assertNetwork(network);

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
