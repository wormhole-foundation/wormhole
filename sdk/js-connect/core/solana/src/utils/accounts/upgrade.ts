import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';

export function deriveUpgradeAuthorityKey(
  wormholeProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress([Buffer.from('upgrade')], wormholeProgramId);
}
