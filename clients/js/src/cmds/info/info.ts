import * as chainId from "./chainId";
import * as contract from "./contract";
import * as emitter from "./emitter";
import * as origin from "./origin";
import * as registrations from "./registrations";
import * as rpc from "./rpc";
import * as wrapped from "./wrapped";

// Commands can be imported as an array of commands.
// Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
export const INFO_COMMANDS = [
  chainId,
  contract,
  emitter,
  origin,
  registrations,
  rpc,
  wrapped,
];
