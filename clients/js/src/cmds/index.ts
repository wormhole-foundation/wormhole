// https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
import * as aptos from "./aptos";
import * as editVaa from "./editVaa";
import * as evm from "./evm";
import * as generate from "./generate";
import * as info from "./info";
import * as near from "./near";
import * as parse from "./parse";
import * as recover from "./recover";
import * as submit from "./submit";
import * as sui from "./sui";
import * as transfer from "./transfer";
import * as verifyVaa from "./verifyVaa";
import * as status from "./status";

// Commands can be imported as an array of commands.
// Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
export const CLI_COMMAND_MODULES = [
  aptos,
  editVaa,
  evm,
  generate,
  info,
  near,
  parse,
  recover,
  submit,
  sui,
  transfer,
  verifyVaa,
  status,
];
