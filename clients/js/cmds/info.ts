import yargs from "yargs";

exports.command = "info";
exports.desc = "Contract, chain and rpc information utilities";
exports.builder = (y: typeof yargs) => {
  // Imports modules logic from root commands, more info here -> https://github.com/yargs/yargs/blob/main/docs/advanced.md#providing-a-command-module
  return y
  .command(require('./chainId'))
  .command(require('./rpc'))
  .command(require('./contractAddress'))
};
