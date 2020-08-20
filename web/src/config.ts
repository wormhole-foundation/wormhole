import {PublicKey} from "@solana/web3.js";

const BRIDGE_ADDRESS = "0xac3eB48829fFC3C37437ce4459cE63F1F4d4E0b4";

const SOLANA_BRIDGE_PROGRAM = new PublicKey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");
const TOKEN_PROGRAM = new PublicKey("TokenSVp5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");


export {
    BRIDGE_ADDRESS,
    TOKEN_PROGRAM,
    SOLANA_BRIDGE_PROGRAM
}
