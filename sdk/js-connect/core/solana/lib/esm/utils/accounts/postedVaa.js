import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function derivePostedVaaKey(wormholeProgramId, hash) {
    return utils.deriveAddress([Buffer.from('PostedVAA'), hash], wormholeProgramId);
}
