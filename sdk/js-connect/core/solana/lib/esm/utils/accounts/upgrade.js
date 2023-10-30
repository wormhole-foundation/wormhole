import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveUpgradeAuthorityKey(wormholeProgramId) {
    return utils.deriveAddress([Buffer.from('upgrade')], wormholeProgramId);
}
