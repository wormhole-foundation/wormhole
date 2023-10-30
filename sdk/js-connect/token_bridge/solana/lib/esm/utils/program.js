import { PublicKey } from '@solana/web3.js';
import { Program } from '@project-serum/anchor';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import { TokenBridgeCoder } from './coder';
import IDL from '../anchor-idl/token_bridge.json';
export function createTokenBridgeProgramInterface(programId, provider) {
    return new Program(IDL, new PublicKey(programId), provider === undefined ? { connection: null } : provider, coder());
}
export function createReadOnlyTokenBridgeProgramInterface(programId, connection) {
    return createTokenBridgeProgramInterface(programId, utils.createReadOnlyProvider(connection));
}
export function coder() {
    return new TokenBridgeCoder(IDL);
}
