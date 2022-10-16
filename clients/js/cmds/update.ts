import { config } from '../config';
import { spawnSync } from 'child_process';

let dir = `${config.wormholeDir}/clients/js`;

exports.command = 'update';
exports.desc = 'Update this tool by rebuilding it';
exports.handler = function(_argv: any) {
    if (isOutdated()) {
        console.log(`Building in ${dir}...`);
        spawnSync(`make build -C ${dir}`, { shell: true, stdio: 'inherit' });
    } else {
        console.log("'worm' is up to date");
    }
}

export function isOutdated(): boolean {
    const result = spawnSync(`make build -C ${dir} --question`, { shell: true });
    return result.status !== 0;
}
