import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveFeeCollectorKey(wormholeProgramId) {
    return utils.deriveAddress([Buffer.from('fee_collector')], wormholeProgramId);
}
