import yargs, { CommandModule } from "yargs";
import * as chainId from "./chainId";
import * as contract from "./contract";
import * as emitter from "./emitter";
import * as origin from "./origin";
import * as registrations from "./registrations";
import * as rpc from "./rpc";
import * as wrapped from "./wrapped";
import { YargsCommandModule } from "../Yargs";

export const command = "info";
export const desc = "Contract, chain, rpc and address information utilities";
// Imports modules logic from root commands, more info here -> https://github.com/yargs/yargs/blob/main/docs/advanced.md#providing-a-command-module
export const builder = (y: typeof yargs) =>
  y
    .command(chainId as unknown as YargsCommandModule)
    .command(contract as unknown as YargsCommandModule)
    .command(emitter as unknown as YargsCommandModule)
    .command(origin as unknown as YargsCommandModule)
    .command(registrations as unknown as YargsCommandModule)
    .command(rpc as unknown as YargsCommandModule)
    .command(wrapped as unknown as YargsCommandModule);
export const handler = () => {};
