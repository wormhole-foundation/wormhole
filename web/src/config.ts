import {PublicKey} from "@solana/web3.js";

const BRIDGE_ADDRESS = "0xdae0Cba01eFc4bfEc1F7Fece73Fe8b8d2Eda65B0";
const WRAPPED_MASTER = "9f7bedd9ef2d57eccab2cb56a5bd395edbb77df8"


const SOLANA_BRIDGE_PROGRAM = new PublicKey("BrdgiFmZN3BKkcY3danbPYyxPKwb8RhQzpM2VY5L97ED");
const TOKEN_PROGRAM = new PublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA");

const SOLANA_HOST = "https://testnet.solana.com";

export {
    BRIDGE_ADDRESS,
    TOKEN_PROGRAM,
    WRAPPED_MASTER,
    SOLANA_BRIDGE_PROGRAM,
    SOLANA_HOST
}
