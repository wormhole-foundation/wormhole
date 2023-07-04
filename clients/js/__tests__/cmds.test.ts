import { describe } from "@jest/globals";
import { test_command_positional_args_with_readme_file } from "./utils/tests";
import * as fs from "fs";
import { CLI_COMMAND_MODULES } from "../src/main";

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

const commandNames = getCommandNamesFromCommandModules(CLI_COMMAND_MODULES);

commandNames.forEach((cmd) => {
  describe(`worm ${cmd}`, () => {
    test_command_positional_args_with_readme_file(cmd, readme);
  });
});
