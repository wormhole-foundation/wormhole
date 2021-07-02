import {PublicKey} from "@solana/web3.js";

const BRIDGE_ADDRESS = "0x254dffcd3277c0b1660f6d42efbb754edababc2b";
const WRAPPED_MASTER = "9A5e27995309a03f8B583feBdE7eF289FcCdC6Ae"


const SOLANA_BRIDGE_PROGRAM = new PublicKey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");
const TOKEN_PROGRAM = new PublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA");

const SOLANA_HOST = "http://localhost:8899";

export {
    BRIDGE_ADDRESS,
    TOKEN_PROGRAM,
    WRAPPED_MASTER,
    SOLANA_BRIDGE_PROGRAM,
    SOLANA_HOST
}
