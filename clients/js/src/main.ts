#!/usr/bin/env node
import yargs from "yargs";
import { hideBin } from "yargs/helpers";
// Quiet is here so that it can trigger before the afflicted libraries' on-import warnings can be emitted.
// It is also imported so that it can side-effect without being tree-shaken.
import "./quiet";
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
