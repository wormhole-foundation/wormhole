import yargs from "yargs";
import {
  CHAINS,
  assertChain,
  coalesceChainId,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";

exports.command = "chain-id <chain>";
exports.desc =
  "Print the wormhole chain ID integer associated with the specified chain name";
exports.builder = (y: typeof yargs) => {
  return y.positional("chain", {
    describe: "Chain to query",
    type: "string",
    choices: Object.keys(CHAINS),
  });
};
exports.handler = (argv) => {
  assertChain(argv["chain"]);
  console.log(coalesceChainId(argv["chain"]));
};
