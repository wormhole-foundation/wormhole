#!/usr/bin/env node

// <sigh>
// when the native secp256k1 is missing, the eccrypto library decides TO PRINT A MESSAGE TO STDOUT:
// https://github.com/bitchan/eccrypto/blob/a4f4a5f85ef5aa1776dfa1b7801cad808264a19c/index.js#L23
//
// do you use a CLI tool that depends on that library and try to pipe the output
// of the tool into another? tough luck
//
// for lack of a better way to stop this, we patch the console.info function to
// drop that particular message...
// </sigh>
const info = console.info;
console.info = function (x: string) {
  if (x != "secp256k1 unavailable, reverting to browser version") {
    info(x);
  }
};

import yargs from "yargs";
import { hideBin } from "yargs/helpers";

yargs(hideBin(process.argv))
  .commandDir("cmds", { recurse: true })
  .strict()
  .demandCommand().argv;
