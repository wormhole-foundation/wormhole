const CONFIG_DIR = `${process.env.HOME}/.wormhole`;
const CONFIG_FILE = `${CONFIG_DIR}/default.json`;

process.env["NODE_CONFIG_DIR"] = CONFIG_DIR;
process.env["SUPPRESS_NO_CONFIG_WARNING"] = "y";
import c from 'config';
import fs from 'fs';

export interface Config {
    // Path to the wormhole repository
    wormholeDir: string;
}

const defaultConfig: Required<Config> = {
    wormholeDir: computeRepoRootPath(),
}

/**
 * Global config object.
 * Importing this module will read the config file and update it if necessary.
 */
export const config: Readonly<Config> = readAndUpdateConfig();

// Computes the path to the root of the wormhole repository based on the
// location of this file (well, the compiled version of this file).
function computeRepoRootPath(): string {
    let rel = "/clients/js/build/config.js";
    // check if mainPath matches $DIR/clients/js/build/config.js
    if (__filename.endsWith(rel)) {
        // if so, grab $DIR from mainPath
        return __filename.substring(0, __filename.length - rel.length);
    } else {
        // otherwise, throw an error
        throw new Error(`Could not compute repo root path for ${__filename}`);
    }
}

function readAndUpdateConfig(): Readonly<Config> {
    if (config !== undefined) {
        return config;
    }
    let conf = defaultConfig;
    // iterate through all the keys in defaultConfig
    for (const key in conf) {
        // if the key is not in config, set it to the default value
        if (c.has(key)) {
            conf[key] = c.get(key);
        }
    }

    let json_conf = JSON.stringify(conf, null, 2) + "\n";

    // if the config file does not exist or does not have some of the default
    // values, create/update it
    let write = false;
    if (!fs.existsSync(CONFIG_FILE)) {
        console.error('\x1b[33m%s\x1b[0m', `NOTE: Created config file at ${CONFIG_FILE}`);
        write = true;
    } else if (json_conf !== fs.readFileSync(CONFIG_FILE, "utf8")) {
        // ^ this will also normalise the config file, but the main thing is
        // that it writes out defaults if they are missing
        console.error('\x1b[33m%s\x1b[0m', `NOTE: Updated config file at ${CONFIG_FILE}`);
        write = true;
    }

    if (write) {
        if (!fs.existsSync(CONFIG_DIR)){
            fs.mkdirSync(CONFIG_DIR, { recursive: true });
        }
        fs.writeFileSync(CONFIG_FILE, json_conf);
    }

    return conf;
}
