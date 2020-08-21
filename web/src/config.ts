import {PublicKey} from "@solana/web3.js";

const BRIDGE_ADDRESS = "0x5b1869D9A4C187F2EAa108f3062412ecf0526b24";

const SOLANA_BRIDGE_PROGRAM = new PublicKey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");
const TOKEN_PROGRAM = new PublicKey("TokenSVp5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");


export {
    BRIDGE_ADDRESS,
    TOKEN_PROGRAM,
    SOLANA_BRIDGE_PROGRAM
}
