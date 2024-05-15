import yargs from "yargs";
import { getEmitterAddress } from "../../emitter";
import { chainToChain } from "../../utils";

export const command = "emitter <chain> <address>";
export const desc = "Print address in emitter address format";
export const builder = (y: typeof yargs) =>
  y
    .positional("chain", {
      describe:
        "Chain to query. To see a list of supported chains, run `worm chains`",
      type: "string",
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
  console.log(await getEmitterAddress(chainToChain(argv.chain), argv.address));
};
