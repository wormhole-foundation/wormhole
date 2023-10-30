import { createApproveInstruction } from '@solana/spl-token';
import { PublicKey } from '@solana/web3.js';
import { deriveAuthoritySignerKey } from '../accounts';
export function createApproveAuthoritySignerInstruction(tokenBridgeProgramId, tokenAccount, owner, amount) {
    return createApproveInstruction(new PublicKey(tokenAccount), deriveAuthoritySignerKey(tokenBridgeProgramId), new PublicKey(owner), amount);
}
