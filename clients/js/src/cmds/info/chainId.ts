import yargs from "yargs";
import { chainToCliChain, cliChainToChainId } from "../../utils";

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
  const inputChain = chainToCliChain(argv.chain);
  console.log(cliChainToChainId(inputChain).toString());
};
