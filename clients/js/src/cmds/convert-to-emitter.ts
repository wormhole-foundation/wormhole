import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import yargs from "yargs";
import { getEmitterAddress } from "../emitter";

export const command = "convert-to-emitter <chain> <address-to-convert>";
export const desc = "Print address in emitter address format";
export const builder = (y: typeof yargs) =>
  y
    .positional("chain", {
      describe: "Chain to query",
      type: "string",
      choices: Object.keys(CHAINS),
      demandOption: true,
    } as const)
    .positional("address-to-convert", {
      describe: "Address to be converted to emitter address format",
      type: "string",
      demandOption: true,
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  assertChain(argv["chain"]);
  console.log(
    await getEmitterAddress(argv["chain"], argv["address-to-convert"])
  );
};
