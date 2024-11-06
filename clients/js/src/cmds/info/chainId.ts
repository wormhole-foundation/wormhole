import yargs from "yargs";
import { chainToChain } from "../../utils";
import { chainToChainId } from "@wormhole-foundation/sdk";

export const command = "chain-id <chain>";
export const desc =
  "Print the wormhole chain ID integer associated with the specified chain name";
export const builder = (y: typeof yargs) => {
  return y.positional("chain", {
    describe:
      "Chain to query. To see a list of supported chains, run `worm chains`",
    type: "string",
    demandOption: true,
  } as const);
};
export const handler = (argv: Awaited<ReturnType<typeof builder>["argv"]>) => {
  const inputChain = chainToChain(argv.chain);
  console.log(chainToChainId(inputChain));
};
