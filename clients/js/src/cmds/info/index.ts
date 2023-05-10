import yargs from "yargs";
import * as chainId from "./chainId";
import * as contract from "./contract";
import * as emitter from "./emitter";
import * as origin from "./origin";
import * as rpc from "./rpc";
import * as wrapped from "./wrapped";

export const command = "info";
export const desc = "Contract, chain, rpc and address information utilities";
// Imports modules logic from root commands, more info here -> https://github.com/yargs/yargs/blob/main/docs/advanced.md#providing-a-command-module
export const builder = (y: typeof yargs) =>
  y
    .command(chainId)
    .command(contract)
    .command(emitter)
    .command(origin)
    .command(rpc)
    .command(wrapped);
export const handler = () => {};
