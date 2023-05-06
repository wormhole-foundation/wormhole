import yargs from "yargs";
import * as chainId from "./chainId";
import * as contractAddress from "./contractAddress";
import * as convertToEmitter from "./convert-to-emitter";
import * as rpc from "./rpc";

export const command = "info";
export const desc = "Contract, chain, rpc and address information utilities";
// Imports modules logic from root commands, more info here -> https://github.com/yargs/yargs/blob/main/docs/advanced.md#providing-a-command-module
export const builder = (y: typeof yargs) =>
  y
    .command(chainId)
    .command(contractAddress)
    .command(convertToEmitter)
    .command(rpc);
export const handler = () => {};
