import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';

export function deriveSenderAccountKey(
  cpiProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress([Buffer.from('sender')], cpiProgramId);
}

export function deriveRedeemerAccountKey(
  cpiProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress([Buffer.from('redeemer')], cpiProgramId);
}
