import yargs from "yargs";
import { YargsCommandModule } from "../Yargs";
import { INFO_COMMANDS } from "./info";

export const command = "info";
export const desc = "Contract, chain, rpc and address information utilities";
// Imports modules logic from root commands, more info here -> https://github.com/yargs/yargs/blob/main/docs/advanced.md#providing-a-command-module
export const builder = (y: typeof yargs) =>
  // Commands can be imported as an array of commands.
  // Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
  y.command(INFO_COMMANDS as unknown as YargsCommandModule[]);
export const handler = () => {};
