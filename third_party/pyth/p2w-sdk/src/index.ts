// import {Connection, PublicKey, SystemProgram} from "@solana/web3.js";
import { ixFromRust} from "@certusone/wormhole-sdk";

async function p2wHello() {
    const p2w = await import("./solana/p2w-core/pyth2wormhole");
    let s = p2w.hello_p2w();
    console.log(s);
}

p2wHello();
