import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveSenderAccountKey(cpiProgramId) {
    return utils.deriveAddress([Buffer.from('sender')], cpiProgramId);
}
export function deriveRedeemerAccountKey(cpiProgramId) {
    return utils.deriveAddress([Buffer.from('redeemer')], cpiProgramId);
}
