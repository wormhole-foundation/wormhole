import { execSync } from "child_process";

// Get machine path targeting to generated `worm` CLI executable, only once
const WORM_CLI_PATH = execSync("which worm", { encoding: "utf8" }).trim();

export const run_worm_command = (commandArgs: string) => {
  const worm = WORM_CLI_PATH;
  return execSync(`${worm} ${commandArgs}`).toString();
};

export const run_worm_help_command = (commandArgs: string) => {
  return run_worm_command(`${commandArgs} --help`);
};
