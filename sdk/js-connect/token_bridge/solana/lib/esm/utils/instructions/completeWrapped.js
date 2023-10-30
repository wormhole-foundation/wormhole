import { PublicKey, SystemProgram, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveEndpointKey, deriveTokenBridgeConfigKey, deriveWrappedMintKey, deriveWrappedMetaKey, deriveMintAuthorityKey, } from '../accounts';
import { toChainId } from '@wormhole-foundation/connect-sdk';
export function createCompleteTransferWrappedInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.completeWrapped();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getCompleteTransferWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getCompleteTransferWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const mint = deriveWrappedMintKey(tokenBridgeProgramId, toChainId(vaa.payload.token.chain), vaa.payload.token.address.toUint8Array());
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        vaa: utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), toChainId(vaa.emitterChain), vaa.sequence),
        endpoint: deriveEndpointKey(tokenBridgeProgramId, toChainId(vaa.emitterChain), vaa.emitterAddress.toUint8Array()),
        to: new PublicKey(vaa.payload.to.address.toUint8Array()),
        toFees: new PublicKey(feeRecipient === undefined
            ? vaa.payload.to.address.toUint8Array()
            : feeRecipient),
        mint,
        wrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
        mintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
