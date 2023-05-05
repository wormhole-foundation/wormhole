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
const infoTemp = console.info;
console.info = function (x: string) {
  if (x != "secp256k1 unavailable, reverting to browser version") {
    infoTemp(x);
  }
};

import yargs from "yargs";
import { hideBin } from "yargs/helpers";
// https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
import * as aptos from "./cmds/aptos";
import * as chainId from "./cmds/chainId";
import * as contractAddress from "./cmds/contractAddress";
import * as editVaa from "./cmds/edit-vaa";
import * as evm from "./cmds/evm";
import * as generate from "./cmds/generate";
import * as info from "./cmds/info";
import * as near from "./cmds/near";
import * as parse from "./cmds/parse";
import * as recover from "./cmds/recover";
import * as rpc from "./cmds/rpc";
import * as submit from "./cmds/submit";
import * as sui from "./cmds/sui";
import * as verifyVaa from "./cmds/verify-vaa";

yargs(hideBin(process.argv))
  // https://github.com/yargs/yargs/blob/main/docs/advanced.md#commanddirdirectory-opts
  // can't use `.commandDir` because bundling + tree-shaking
  .command(aptos)
  .command(chainId)
  .command(contractAddress)
  .command(editVaa)
  .command(evm)
  .command(generate)
  .command(info)
  .command(near)
  .command(parse)
  .command(recover)
  .command(rpc)
  .command(submit)
  .command(sui)
  .command(verifyVaa)
  .strict()
  .demandCommand().argv;
