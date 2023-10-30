import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveAuthoritySignerKey(tokenBridgeProgramId) {
    return utils.deriveAddress([Buffer.from('authority_signer')], tokenBridgeProgramId);
}
export function deriveCustodySignerKey(tokenBridgeProgramId) {
    return utils.deriveAddress([Buffer.from('custody_signer')], tokenBridgeProgramId);
}
export function deriveMintAuthorityKey(tokenBridgeProgramId) {
    return utils.deriveAddress([Buffer.from('mint_signer')], tokenBridgeProgramId);
}
