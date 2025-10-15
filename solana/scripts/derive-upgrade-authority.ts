import { PublicKey } from "@solana/web3.js";
import { Buffer } from "buffer";

const programId = process.env.PROGRAM_ID;

if (!programId) {
    console.error("PROGRAM_ID environment variable not set.");
    process.exit(1);
}

console.log(
    PublicKey.findProgramAddressSync([Buffer.from("upgrade")], new PublicKey(programId))[0].toString()
);
