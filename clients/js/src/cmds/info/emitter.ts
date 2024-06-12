import yargs from "yargs";
import { getEmitterAddress } from "../../emitter";
import { assertChain, chains } from "@wormhole-foundation/sdk-base";

export const command = "emitter <chain> <address>";
export const desc = "Print address in emitter address format";
export const builder = (y: typeof yargs) =>
  y
    .positional("chain", {
      describe: "Chain to query",
      type: "string",
      choices: chains,
      demandOption: true,
    } as const)
    .positional("address", {
      describe: "Address to be converted to emitter address format",
      type: "string",
      demandOption: true,
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  assertChain(argv.chain);
  console.log(await getEmitterAddress(argv.chain, argv.address));
};
