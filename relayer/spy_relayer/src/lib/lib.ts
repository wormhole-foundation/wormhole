import { exec } from 'child_process'
import * as fs from 'fs'

export async function shell(cmd: string): Promise<string> {
    return new Promise((resolve, reject) => {
        exec(cmd, (err, stdout, stderr) => err ? reject(err) : resolve(stdout.trim()))
    })
}

export async function rootPath(): Promise<string> {
    return await shell("git rev-parse --show-toplevel")
}

export async function loadAddrs(): Promise<any> {
    const dir = await rootPath()
    return JSON.parse(String(await fs.promises.readFile(dir + "/addrs")));
}


