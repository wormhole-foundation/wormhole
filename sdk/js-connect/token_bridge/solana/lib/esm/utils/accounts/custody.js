import { PublicKey } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveCustodyKey(tokenBridgeProgramId, mint) {
    return utils.deriveAddress([new PublicKey(mint).toBuffer()], tokenBridgeProgramId);
}
