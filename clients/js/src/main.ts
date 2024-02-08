#!/usr/bin/env node
import yargs from "yargs";
import { hideBin } from "yargs/helpers";
import { YargsCommandModule } from "./cmds/Yargs";
import { CLI_COMMAND_MODULES } from "./cmds";

// Side effects are here to trigger before the affected libraries' on-import warnings can be emitted.
import "./side-effects";

const argv = yargs(hideBin(process.argv))
  // Build CLI commands dynamically from CLI_COMMAND_MODULES list
  // Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
  .command(CLI_COMMAND_MODULES as YargsCommandModule[])
  .strict()
  .demandCommand()
  .argv;
