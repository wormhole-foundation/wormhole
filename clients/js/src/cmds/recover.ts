import { ethers } from "ethers";
import yargs from "yargs";
import { hex } from "../utils";

export const command = "recover <digest> <signature>";
export const desc = "Recover an address from a signature";
export const builder = (y: typeof yargs) => {
  return y
    .positional("digest", {
      describe: "digest",
      type: "string",
    })
    .positional("signature", {
      describe: "signature",
      type: "string",
    });
};
export const handler = async (argv) => {
  console.log(
    ethers.utils.recoverAddress(hex(argv["digest"]), hex(argv["signature"]))
  );
};
