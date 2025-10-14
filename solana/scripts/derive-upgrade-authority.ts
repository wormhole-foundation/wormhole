import { PublicKey } from "@solana/web3.js";
import { Buffer } from "buffer";

const tokenBridgeProgramId = process.env.TOKEN_BRIDGE_PROGRAM_ID;

if (!tokenBridgeProgramId) {
    console.error("TOKEN_BRIDGE_PROGRAM_ID environment variable not set.");
    process.exit(1);
}

console.log(
    PublicKey.findProgramAddressSync([Buffer.from("upgrade")], new PublicKey(tokenBridgeProgramId))[0].toString()
);
