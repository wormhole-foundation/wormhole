import { PublicKey, SystemProgram, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveEndpointKey, deriveTokenBridgeConfigKey, deriveCustodyKey, deriveCustodySignerKey, } from '../accounts';
import { toChainId } from '@wormhole-foundation/connect-sdk';
export function createCompleteTransferNativeInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.completeNative();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getCompleteTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getCompleteTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const mint = new PublicKey(vaa.payload.token.address.toUint8Array());
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
        custody: deriveCustodyKey(tokenBridgeProgramId, mint),
        mint,
        custodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
