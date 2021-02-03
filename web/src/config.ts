import {PublicKey} from "@solana/web3.js";

const BRIDGE_ADDRESS = "0xf92cD566Ea4864356C5491c177A430C222d7e678";
const WRAPPED_MASTER = "9A5e27995309a03f8B583feBdE7eF289FcCdC6Ae"


const SOLANA_BRIDGE_PROGRAM = new PublicKey("WormT3McKhFJ2RkiGpdw9GKvNCrB2aB54gb2uV9MfQC");
const TOKEN_PROGRAM = new PublicKey("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA");

const SOLANA_HOST = "https://solana-api.projectserum.com";

export {
    BRIDGE_ADDRESS,
    TOKEN_PROGRAM,
    WRAPPED_MASTER,
    SOLANA_BRIDGE_PROGRAM,
    SOLANA_HOST
}
