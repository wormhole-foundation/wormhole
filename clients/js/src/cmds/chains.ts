import yargs from "yargs";
import { chains } from "@wormhole-foundation/sdk-base";

export const command = "chains";
export const desc = "Print the list of supported chains";
export const builder = (y: typeof yargs) => {
  // No positional parameters needed
  return y;
};
export const handler = () => {
  console.log(chains);
};
