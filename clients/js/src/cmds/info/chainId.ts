import {
  CHAINS,
  assertChain,
  coalesceChainId,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";

export const command = "chain-id <chain>";
export const desc =
  "Print the wormhole chain ID integer associated with the specified chain name";
export const builder = (y: typeof yargs) => {
  return y.positional("chain", {
    describe: "Chain to query",
    type: "string",
    choices: Object.keys(CHAINS) as (keyof typeof CHAINS)[],
    demandOption: true,
  } as const);
};
export const handler = (argv: Awaited<ReturnType<typeof builder>["argv"]>) => {
  assertChain(argv["chain"]);
  console.log(coalesceChainId(argv["chain"]));
};
