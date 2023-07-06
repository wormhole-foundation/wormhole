import { describe, it, expect } from "@jest/globals";
import * as fs from "fs";
import { CLI_COMMAND_MODULES } from "../src/cmds";
import { run_worm_help_command } from "./utils/cli";

const readme = fs.readFileSync("./README.md", "utf8");

const getCommandNamesFromCommandModules = (
  cmdModules: typeof CLI_COMMAND_MODULES
) => {
  return cmdModules
    .map((cmdModule) => cmdModule.command)
    .map(
      (commandStr) => commandStr.split("<")[0].trim() //Removing <positional> arguments from commands module strings
    );
};

const getCommandWithArgsFromOutput = (rawOutput: string) => {
  return rawOutput.split("worm")[1].split("\n")[0].trim();
};

const commandNames = getCommandNamesFromCommandModules(CLI_COMMAND_MODULES);

commandNames.forEach((cmd) => {
  describe(`worm ${cmd}`, () => {
    it(`should have same command args as documentation`, async () => {
      // Run the command module with --help as argument
      const output = run_worm_help_command(cmd);
      const commandWithArgs = getCommandWithArgsFromOutput(output);
  
      expect(readme).toContain(commandWithArgs);
    });
  });
});
