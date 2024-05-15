import yargs from "yargs";
import { CHAIN_ID_OR_NAME_CHOICES } from "../../consts";
import { assertChain, toChain } from "@wormhole-foundation/sdk-base";

export const command = "chain-id <chain>";
export const desc =
  "Print the wormhole chain ID integer associated with the specified chain name";
export const builder = (y: typeof yargs) => {
  return y.positional("chain", {
    describe: "Chain to query",
    choices: CHAIN_ID_OR_NAME_CHOICES,
    demandOption: true,
  } as const);
};
export const handler = (argv: Awaited<ReturnType<typeof builder>["argv"]>) => {
  assertChain(toChain(argv.chain));
  console.log(toChain(argv.chain));
};
