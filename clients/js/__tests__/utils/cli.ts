import { execSync } from "child_process";

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
