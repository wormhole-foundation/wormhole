import yargs from "yargs";
import {
  CHAINS,
  assertChain,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { getEmitterAddress } from "../emitter";

exports.command = "convert-to-emitter <chain> <address-to-convert>";
exports.desc = "Print address in emitter address format";
exports.builder = (y: typeof yargs) => {
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
exports.handler = async (argv) => {
  assertChain(argv["chain"]);
  let chain = argv["chain"];
  console.log(await getEmitterAddress(chain, argv["address-to-convert"]));
};
