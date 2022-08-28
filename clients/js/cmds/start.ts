import yargs from "yargs";
import { spawnSync } from 'child_process';
import { config } from '../config';

exports.command = 'start-validator';
exports.desc = 'Start a local validator';
exports.builder = function(y: typeof yargs) {
    return y.option("validator-args", {
        alias: "a",
        type: "string",
        array: true,
        default: [],
        describe: "Additional args to validator",
    }).command("aptos", "Start a local aptos validator", (yargs) => {
    }, (argv) => {
        const dir = `${config.wormholeDir}/aptos`;
        // check if aptos is installed
        const aptos = spawnSync("aptos", ["--version"]);
        if (aptos.status !== 0) {
            console.error("aptos is not installed. Please install aptos and try again.");
            console.error(`See ${dir}/README.md for instructions.`);
            process.exit(1);
        }
        const args = argv['validator-args'].map(a => `"${a}"`).join(" ");
        const cmd = `cd ${dir} && aptos node run-local-testnet --with-faucet --force-restart --assume-yes ${args}`;
        console.log("\x1b[33m%s\x1b[0m", cmd);
        spawnSync(cmd, { shell: true, stdio: "inherit" });
    }).strict().demandCommand();
}
