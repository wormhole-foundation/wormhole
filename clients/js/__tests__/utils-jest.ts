import { execSync } from "child_process";
import { expect, it } from "@jest/globals";

export const get_WORM_CLI_PATH = () => {
  //This functions returns the machine path targeting to the generated `worm` command
  return execSync("which worm", { encoding: "utf8" }).trim();
};

export const run_worm_command = (commandArgs: string) => {
  const worm = get_WORM_CLI_PATH();
  return execSync(`${worm} ${commandArgs}`).toString();
};

export const run_worm_help_command = (commandArgs: string) => {
  return run_worm_command(`${commandArgs} --help`);
};

export const test_command_positional_args = (
  command: string,
  args: string[]
) => {
  //NOTE: Guard condition to avoid passing infered `worm` keyword from command input
  if (command.includes("worm")) {
    throw new Error(
      "initial 'worm' keyword must be excluded from command params, pass only worm specific commands."
    );
  }

  it(`should have correct positional arguments`, async () => {
    // Run the command module with --help as argument
    const output = run_worm_help_command(command);
    const expectedPositionalArgs = args.map((arg) => `<${arg}>`).join(" ");

    expect(output).toContain(`worm ${command} ${expectedPositionalArgs}`);
  });
};
