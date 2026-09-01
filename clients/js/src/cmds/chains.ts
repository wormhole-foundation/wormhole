import yargs from "yargs";
import { CLI_CHAINS } from "../utils";

export const command = "chains";
export const desc = "Print the list of supported chains";
export const builder = (y: typeof yargs) => {
  // No positional parameters needed
  return y;
};
export const handler = () => {
  console.log(CLI_CHAINS);
};
