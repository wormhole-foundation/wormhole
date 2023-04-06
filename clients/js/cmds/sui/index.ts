import yargs from "yargs";
import { Yargs } from "../Yargs";
import { addDeployCommands } from "./deploy";
import { addInitCommands } from "./init";
import { addPublishMessageCommands } from "./publish_message";
import { addUtilsCommands } from "./utils";

exports.command = "sui";
exports.desc = "Sui utilities";
exports.builder = function (y: typeof yargs) {
  return new Yargs(y)
    .addCommands(addDeployCommands)
    .addCommands(addInitCommands)
    .addCommands(addPublishMessageCommands)
    .addCommands(addUtilsCommands)
    .y()
    .strict()
    .demandCommand();
};
