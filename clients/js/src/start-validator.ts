import { spawnSync } from 'child_process';

export const validator_args = {
  alias: "a",
  type: "string",
  array: true,
  default: [],
  describe: "Additional args to validator",
} as const;

export function runCommand(baseCmd: string, args: readonly string[]) {
  const args_string = args.map(a => `"${a}"`).join(" ");
  const cmd = `${baseCmd} ${args_string}`;
  console.log("\x1b[33m%s\x1b[0m", cmd);
  spawnSync(cmd, { shell: true, stdio: "inherit" });
}
