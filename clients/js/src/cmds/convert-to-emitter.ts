import yargs from "yargs";
import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { getEmitterAddress } from "../emitter";

export const command = "convert-to-emitter <chain> <address-to-convert>";
export const desc = "Print address in emitter address format";
export const builder = (y: typeof yargs) => {
  return y
    .positional("chain", {
      describe: "Chain to query",
      type: "string",
      choices: Object.keys(CHAINS),
    })
    .positional("address-to-convert", {
      describe: "Address to be converted to emitter address format",
      type: "string",
    });
};
export const handler = async (argv) => {
  assertChain(argv["chain"]);
  let chain = argv["chain"];
  console.log(await getEmitterAddress(chain, argv["address-to-convert"]));
};
