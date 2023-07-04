import { expect, it } from "@jest/globals";
import { run_worm_help_command } from "./cli";

export const test_command_args_with_readme_file = (
  command: string,
  readmeFileContent: string
) => {
  //NOTE: Guard condition to avoid passing infered `worm` keyword from command input
  const wormCommandRegex = /^worm /;
  if (new RegExp(wormCommandRegex).test(command)) {
    throw new Error(
      "initial 'worm' keyword must be excluded from command params, pass only worm specific commands."
    );
  }

  it(`should have same command args as documentation`, async () => {
    // Run the command module with --help as argument
    const output = run_worm_help_command(command);

    const getCmdFromOutput = (rawOutput: string) => {
      return rawOutput.split("worm")[1].split("\n")[0].trim();
    };
    const cmd = getCmdFromOutput(output);

    expect(readmeFileContent).toContain(cmd);
  });
};
