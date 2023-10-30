import { PublicKey } from '@solana/web3.js';
import { Program } from '@project-serum/anchor';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import { WormholeCoder } from './coder';
import IDL from '../anchor-idl/wormhole.json';
export function createWormholeProgramInterface(programId, provider) {
    return new Program(IDL, new PublicKey(programId), provider === undefined ? { connection: null } : provider, coder());
}
export function createReadOnlyWormholeProgramInterface(programId, connection) {
    return createWormholeProgramInterface(programId, utils.createReadOnlyProvider(connection));
}
export function coder() {
    return new WormholeCoder(IDL);
}
