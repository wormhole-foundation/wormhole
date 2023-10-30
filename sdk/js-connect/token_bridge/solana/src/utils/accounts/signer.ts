import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';

export function deriveAuthoritySignerKey(
  tokenBridgeProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress(
    [Buffer.from('authority_signer')],
    tokenBridgeProgramId,
  );
}

export function deriveCustodySignerKey(
  tokenBridgeProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress(
    [Buffer.from('custody_signer')],
    tokenBridgeProgramId,
  );
}

export function deriveMintAuthorityKey(
  tokenBridgeProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress(
    [Buffer.from('mint_signer')],
    tokenBridgeProgramId,
  );
}
