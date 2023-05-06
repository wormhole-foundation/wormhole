import { ethers } from "ethers";
import yargs from "yargs";
import { hex } from "../utils";

export const command = "recover <digest> <signature>";
export const desc = "Recover an address from a signature";
export const builder = (y: typeof yargs) =>
  y
    .positional("digest", {
      describe: "digest",
      type: "string",
      demandOption: true,
    })
    .positional("signature", {
      describe: "signature",
      type: "string",
      demandOption: true,
    });

export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  console.log(
    ethers.utils.recoverAddress(hex(argv.digest), hex(argv.signature))
  );
};
