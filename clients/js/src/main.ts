#!/usr/bin/env node
import yargs from "yargs";
import { hideBin } from "yargs/helpers";
// Side effects are here to trigger before the afflicted libraries' on-import warnings can be emitted.
// It is also imported so that it can side-effect without being tree-shaken.
import "./side-effects";
import { YargsCommandModule } from "./cmds/Yargs";
import { CLI_COMMAND_MODULES } from "./cmds";

yargs(hideBin(process.argv))
  // Build CLI commands dinamically from CLI_COMMAND_MODULES list
  .command(CLI_COMMAND_MODULES as YargsCommandModule[])
  .strict()
  .demandCommand().argv;
