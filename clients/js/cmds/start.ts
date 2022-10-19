import yargs from "yargs";
import { spawnSync } from 'child_process';
import { config } from '../config';
import { checkAptosBinary } from "./aptos";

exports.command = 'start-validator';
exports.desc = 'Start a local validator';
exports.builder = function(y: typeof yargs) {
    return y.option("validator-args", {
        alias: "a",
        type: "string",
        array: true,
        default: [],
        describe: "Additional args to validator",
    }).command("aptos", "Start a local aptos validator", (_yargs) => {
    }, (argv) => {
        const dir = `${config.wormholeDir}/aptos`;
        checkAptosBinary();
        const cmd = `cd ${dir} && aptos node run-local-testnet --with-faucet --force-restart --assume-yes`;
        runCommand(cmd, argv['validator-args']);
    }).command("evm", "Start a local EVM validator", (_yargs) => {
    }, (argv) => {
        const dir = `${config.wormholeDir}/ethereum`;
        const cmd = `cd ${dir} && npx ganache-cli -e 10000 --deterministic --time="1970-01-01T00:00:00+00:00"`;
        runCommand(cmd, argv['validator-args']);
    }).strict().demandCommand();
}

function runCommand(baseCmd: string, args: string[]) {
    const args_string = args.map(a => `"${a}"`).join(" ");
    const cmd = `${baseCmd} ${args_string}`;
    console.log("\x1b[33m%s\x1b[0m", cmd);
    spawnSync(cmd, { shell: true, stdio: "inherit" });
}
