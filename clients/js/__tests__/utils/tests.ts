import { expect, it } from "@jest/globals";
import { run_worm_help_command } from "./cli";

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

export type Flag = {
  name: string;
  alias?: string;
};

export const test_command_flags = (command: string, flags: Flag[]) => {
  const wormCommandRegex = /^worm /;

  //NOTE: Guard condition to avoid passing infered `worm` keyword from command input
  if (new RegExp(wormCommandRegex).test(command)) {
    throw new Error(
      "initial 'worm' keyword must be excluded from command params, pass only worm specific commands."
    );
  }
  // Run the command module with --help as argument
  const output = run_worm_help_command(command);

  it(`should have correct flags`, async () => {
    const expectedFlags = flags.map((arg) => arg.name);

    expectedFlags.forEach((flag) => {
      expect(output).toContain(flag);
    });
  });

  it(`should have correct flag alias`, async () => {
    const expectedFlagAlias = flags.map((arg) => arg.alias);

    expectedFlagAlias.forEach((alias) => {
      if (alias) expect(output).toContain(alias);
    });
  });
};
