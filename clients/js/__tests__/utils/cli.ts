import { execSync } from "child_process";
import { expect, it } from "@jest/globals";

// WORM_CLI singleton, this avoids calling getWormCLI on the target machine more than once, is a syncronous costly process.
class WormCLI {
  static instance: WormCLI;
  private path!: string;

  constructor() {
    if (!WormCLI.instance) {
      this.path = this.getWormCLI();
      WormCLI.instance = this;
    }
    //NOTE: guard to send the same instance in case 'new WormCLI()' is called again
    return WormCLI.instance;
  }

  private getWormCLI = (): string => {
    //This functions returns the machine path targeting to the generated `worm` command
    return execSync("which worm", { encoding: "utf8" }).trim();
  };

  getPath = (): string => {
    return this.path;
  };
}

// Init WORM_CLI singleton
const WORM_CLI = new WormCLI();

export const run_worm_command = (commandArgs: string) => {
  const worm = WORM_CLI.getPath();
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

export type Flag = {
  name: string;
  alias?: string;
};

export const test_command_flags = (command: string, flags: Flag[]) => {
  //NOTE: Guard condition to avoid passing infered `worm` keyword from command input
  if (command.includes("worm")) {
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
