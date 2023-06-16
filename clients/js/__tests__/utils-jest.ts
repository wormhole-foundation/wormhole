import { execSync } from "child_process";

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
