#!/usr/bin/env node
import yargs from "yargs";
import { hideBin } from "yargs/helpers";
// Side effects are here to trigger before the afflicted libraries' on-import warnings can be emitted.
// It is also imported so that it can side-effect without being tree-shaken.
import "./side-effects";
import { YargsCommandModule } from "./cmds/Yargs";
import { CLI_COMMAND_MODULES } from "./cmds";

yargs(hideBin(process.argv))
  // Build CLI commands dynamically from CLI_COMMAND_MODULES list
  // Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
  .command(CLI_COMMAND_MODULES as YargsCommandModule[])
  .strict()
  .demandCommand().argv;
