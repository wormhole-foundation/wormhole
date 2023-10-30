"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SolanaTokenBridge = void 0;
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const utils_1 = require("./utils");
class SolanaTokenBridge {
    constructor(network, chain, connection, contracts) {
        this.network = network;
        this.chain = chain;
        this.connection = connection;
        this.contracts = contracts;
        this.chainId = (0, connect_sdk_1.toChainId)(chain);
        const tokenBridgeAddress = contracts.tokenBridge;
        if (!tokenBridgeAddress)
            throw new Error(`TokenBridge contract Address for chain ${chain} not found`);
        this.tokenBridgeAddress = tokenBridgeAddress;
        const coreBridgeAddress = contracts.coreBridge;
        if (!coreBridgeAddress)
            throw new Error(`CoreBridge contract Address for chain ${chain} not found`);
        this.coreAddress = coreBridgeAddress;
        this.tokenBridge = (0, utils_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeAddress, connection);
    }
    static async fromProvider(connection, config) {
        const [network, chain] = await connect_sdk_solana_1.SolanaPlatform.chainFromRpc(connection);
        return new SolanaTokenBridge(network, chain, connection, config[chain].contracts);
    }
    async isWrappedAsset(token) {
        return (0, utils_1.getWrappedMeta)(this.connection, this.tokenBridge.programId, new connect_sdk_solana_1.SolanaAddress(token).toUint8Array())
            .catch((_) => null)
            .then((meta) => meta != null);
    }
    async getOriginalAsset(token) {
        if (!(await this.isWrappedAsset(token)))
            throw (0, connect_sdk_1.ErrNotWrapped)(token.toString());
        const tokenAddr = new connect_sdk_solana_1.SolanaAddress(token).toUint8Array();
        const mint = new web3_js_1.PublicKey(tokenAddr);
        try {
            const meta = await (0, utils_1.getWrappedMeta)(this.connection, this.tokenBridge.programId, tokenAddr);
            if (meta === null)
                return {
                    chain: this.chain,
                    address: (0, connect_sdk_1.toNative)(this.chain, mint.toBytes()),
                };
            return {
                chain: (0, connect_sdk_1.toChainName)(meta.chain),
                address: new connect_sdk_1.UniversalAddress(meta.tokenAddress),
            };
        }
        catch (_) {
            // TODO: https://github.com/wormhole-foundation/wormhole/blob/main/sdk/js/src/token_bridge/getOriginalAsset.ts#L200
            // the current one returns 0s for address
            throw (0, connect_sdk_1.ErrNotWrapped)(token.toString());
        }
    }
    async hasWrappedAsset(token) {
        try {
            await this.getWrappedAsset(token);
            return true;
        }
        catch (_) { }
        return false;
    }
    async getWrappedAsset(token) {
        const mint = (0, utils_1.deriveWrappedMintKey)(this.tokenBridge.programId, (0, connect_sdk_1.toChainId)(token.chain), token.address.toUniversalAddress().toUint8Array());
        // If we don't throw an error getting wrapped meta, we're good to return
        // the derived mint address back to the caller.
        try {
            await (0, utils_1.getWrappedMeta)(this.connection, this.tokenBridge.programId, mint);
            return (0, connect_sdk_1.toNative)(this.chain, mint.toBase58());
        }
        catch (_) { }
        throw (0, connect_sdk_1.ErrNotWrapped)(token.address.toUniversalAddress().toString());
    }
    async isTransferCompleted(vaa) {
        return wormhole_connect_sdk_core_solana_1.utils
            .getClaim(this.connection, this.tokenBridge.programId, vaa.emitterAddress.toUint8Array(), (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.sequence, this.connection.commitment)
            .catch((e) => false);
    }
    async getWrappedNative() {
        return (0, connect_sdk_1.toNative)(this.chain, spl_token_1.NATIVE_MINT.toBase58());
    }
    async *createAttestation(token, payer) {
        if (!payer)
            throw new Error('Payer required to create attestation');
        const senderAddress = new connect_sdk_solana_1.SolanaAddress(payer).unwrap();
        // TODO:
        const nonce = 0; // createNonce().readUInt32LE(0);
        const transferIx = await wormhole_connect_sdk_core_solana_1.utils.createBridgeFeeTransferInstruction(this.connection, this.coreAddress, senderAddress);
        const messageKey = web3_js_1.Keypair.generate();
        const attestIx = (0, utils_1.createAttestTokenInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, new connect_sdk_solana_1.SolanaAddress(token).toUint8Array(), messageKey.publicKey, nonce);
        const transaction = new web3_js_1.Transaction().add(transferIx, attestIx);
        const { blockhash } = await this.connection.getLatestBlockhash();
        transaction.recentBlockhash = blockhash;
        transaction.feePayer = senderAddress;
        transaction.partialSign(messageKey);
        yield this.createUnsignedTx(transaction, 'Solana.AttestToken');
    }
    async *submitAttestation(vaa, payer) {
        if (!payer)
            throw new Error('Payer required to create attestation');
        const senderAddress = new connect_sdk_solana_1.SolanaAddress(payer).unwrap();
        const { blockhash } = await this.connection.getLatestBlockhash();
        // Yield transactions to verify sigs and post the VAA
        yield* this.postVaa(senderAddress, vaa, blockhash);
        // Now yield the transaction to actually create the token
        const transaction = new web3_js_1.Transaction().add((0, utils_1.createCreateWrappedInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, vaa));
        transaction.recentBlockhash = blockhash;
        transaction.feePayer = senderAddress;
        yield this.createUnsignedTx(transaction, 'Solana.CreateWrapped');
    }
    async transferSol(sender, recipient, amount, payload) {
        //  https://github.com/wormhole-foundation/wormhole-connect/blob/development/sdk/src/contexts/solana/context.ts#L245
        const senderAddress = new connect_sdk_solana_1.SolanaAddress(sender).unwrap();
        // TODO: the payer can actually be different from the sender. We need to allow the user to pass in an optional payer
        const payerPublicKey = senderAddress;
        const recipientAddress = recipient.address
            .toUniversalAddress()
            .toUint8Array();
        const recipientChainId = (0, connect_sdk_1.toChainId)(recipient.chain);
        const nonce = 0;
        const relayerFee = 0n;
        const message = web3_js_1.Keypair.generate();
        const ancillaryKeypair = web3_js_1.Keypair.generate();
        const rentBalance = await (0, spl_token_1.getMinimumBalanceForRentExemptAccount)(this.connection);
        //This will create a temporary account where the wSOL will be created.
        const createAncillaryAccountIx = web3_js_1.SystemProgram.createAccount({
            fromPubkey: payerPublicKey,
            newAccountPubkey: ancillaryKeypair.publicKey,
            lamports: rentBalance,
            space: spl_token_1.ACCOUNT_SIZE,
            programId: spl_token_1.TOKEN_PROGRAM_ID,
        });
        //Send in the amount of SOL which we want converted to wSOL
        const initialBalanceTransferIx = web3_js_1.SystemProgram.transfer({
            fromPubkey: payerPublicKey,
            lamports: amount,
            toPubkey: ancillaryKeypair.publicKey,
        });
        //Initialize the account as a WSOL account, with the original payerAddress as owner
        const initAccountIx = (0, spl_token_1.createInitializeAccountInstruction)(ancillaryKeypair.publicKey, spl_token_1.NATIVE_MINT, payerPublicKey);
        //Normal approve & transfer instructions, except that the wSOL is sent from the ancillary account.
        const approvalIx = (0, utils_1.createApproveAuthoritySignerInstruction)(this.tokenBridge.programId, ancillaryKeypair.publicKey, payerPublicKey, amount);
        const tokenBridgeTransferIx = payload
            ? (0, utils_1.createTransferNativeWithPayloadInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, ancillaryKeypair.publicKey, spl_token_1.NATIVE_MINT, nonce, amount, recipientAddress, recipientChainId, payload)
            : (0, utils_1.createTransferNativeInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, ancillaryKeypair.publicKey, spl_token_1.NATIVE_MINT, nonce, amount, relayerFee, recipientAddress, recipientChainId);
        //Close the ancillary account for cleanup. Payer address receives any remaining funds
        const closeAccountIx = (0, spl_token_1.createCloseAccountInstruction)(ancillaryKeypair.publicKey, //account to close
        payerPublicKey, //Remaining funds destination
        payerPublicKey);
        const { blockhash } = await this.connection.getLatestBlockhash();
        const transaction = new web3_js_1.Transaction();
        transaction.recentBlockhash = blockhash;
        transaction.feePayer = payerPublicKey;
        transaction.add(createAncillaryAccountIx, initialBalanceTransferIx, initAccountIx, approvalIx, tokenBridgeTransferIx, closeAccountIx);
        transaction.partialSign(message, ancillaryKeypair);
        return this.createUnsignedTx(transaction, 'Solana.TransferNative');
    }
    async *transfer(sender, recipient, token, amount, payload) {
        // TODO: payer vs sender?? can caller add diff payer later?
        if (token === 'native') {
            yield await this.transferSol(sender, recipient, amount, payload);
            return;
        }
        const tokenAddress = new connect_sdk_solana_1.SolanaAddress(token).unwrap();
        const senderAddress = new connect_sdk_solana_1.SolanaAddress(sender).unwrap();
        const senderTokenAddress = await (0, spl_token_1.getAssociatedTokenAddress)(tokenAddress, senderAddress);
        const recipientAddress = recipient.address
            .toUniversalAddress()
            .toUint8Array();
        const recipientChainId = (0, connect_sdk_1.toChainId)(recipient.chain);
        const nonce = 0;
        const relayerFee = 0n;
        const isSolanaNative = !(await this.isWrappedAsset(token));
        const message = web3_js_1.Keypair.generate();
        let tokenBridgeTransferIx;
        if (isSolanaNative) {
            tokenBridgeTransferIx = payload
                ? (0, utils_1.createTransferNativeWithPayloadInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, senderTokenAddress, tokenAddress, nonce, amount, recipientAddress, recipientChainId, payload)
                : (0, utils_1.createTransferNativeInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, senderTokenAddress, tokenAddress, nonce, amount, relayerFee, recipientAddress, recipientChainId);
        }
        else {
            const originAsset = await this.getOriginalAsset(token);
            tokenBridgeTransferIx = payload
                ? (0, utils_1.createTransferWrappedWithPayloadInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, senderTokenAddress, senderAddress, (0, connect_sdk_1.toChainId)(originAsset.chain), originAsset.address.toUint8Array(), nonce, amount, recipientAddress, recipientChainId, payload)
                : (0, utils_1.createTransferWrappedInstruction)(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, message.publicKey, senderTokenAddress, senderAddress, (0, connect_sdk_1.toChainId)(originAsset.chain), originAsset.address.toUint8Array(), nonce, amount, relayerFee, recipientAddress, recipientChainId);
        }
        const approvalIx = (0, utils_1.createApproveAuthoritySignerInstruction)(this.tokenBridge.programId, senderTokenAddress, senderAddress, amount);
        const transaction = new web3_js_1.Transaction().add(approvalIx, tokenBridgeTransferIx);
        const { blockhash } = await this.connection.getLatestBlockhash();
        transaction.recentBlockhash = blockhash;
        transaction.feePayer = senderAddress;
        transaction.partialSign(message);
        yield this.createUnsignedTx(transaction, 'Solana.TransferTokens');
    }
    async *redeem(sender, vaa, unwrapNative = true) {
        // TODO unwrapNative?
        // TODO: check if vaa.payload.token.address is native Sol
        const { blockhash } = await this.connection.getLatestBlockhash();
        const senderAddress = new connect_sdk_solana_1.SolanaAddress(sender).unwrap();
        const ataAddress = new connect_sdk_solana_1.SolanaAddress(vaa.payload.to.address.toUint8Array()).unwrap();
        const wrappedToken = await this.getWrappedAsset(vaa.payload.token);
        // If the ata doesn't exist yet, create it
        const acctInfo = await this.connection.getAccountInfo(ataAddress);
        if (acctInfo === null) {
            const ataCreationTx = new web3_js_1.Transaction().add((0, spl_token_1.createAssociatedTokenAccountInstruction)(senderAddress, ataAddress, senderAddress, new web3_js_1.PublicKey(wrappedToken.toUint8Array())));
            ataCreationTx.feePayer = senderAddress;
            ataCreationTx.recentBlockhash = blockhash;
            yield this.createUnsignedTx(ataCreationTx, 'Redeem.CreateATA');
        }
        // Yield transactions to verify sigs and post the VAA
        yield* this.postVaa(sender, vaa, blockhash);
        const createCompleteTransferInstruction = vaa.payload.token.chain == this.chain
            ? utils_1.createCompleteTransferNativeInstruction
            : utils_1.createCompleteTransferWrappedInstruction;
        const transaction = new web3_js_1.Transaction().add(createCompleteTransferInstruction(this.connection, this.tokenBridge.programId, this.coreAddress, senderAddress, vaa));
        transaction.recentBlockhash = blockhash;
        transaction.feePayer = senderAddress;
        yield this.createUnsignedTx(transaction, 'Solana.RedeemTransfer');
    }
    async *postVaa(sender, vaa, blockhash) {
        const senderAddr = new connect_sdk_solana_1.SolanaAddress(sender).unwrap();
        const signatureSet = web3_js_1.Keypair.generate();
        const verifySignaturesInstructions = await wormhole_connect_sdk_core_solana_1.utils.createVerifySignaturesInstructions(this.connection, this.coreAddress, senderAddr, vaa, signatureSet.publicKey);
        // Create a new transaction for every 2 signatures we have to Verify
        for (let i = 0; i < verifySignaturesInstructions.length; i += 2) {
            const verifySigTx = new web3_js_1.Transaction().add(...verifySignaturesInstructions.slice(i, i + 2));
            verifySigTx.recentBlockhash = blockhash;
            verifySigTx.feePayer = senderAddr;
            verifySigTx.partialSign(signatureSet);
            yield this.createUnsignedTx(verifySigTx, 'Redeem.VerifySignature', true);
        }
        // Finally create the VAA posting transaction
        const postVaaTx = new web3_js_1.Transaction().add(wormhole_connect_sdk_core_solana_1.utils.createPostVaaInstruction(this.connection, this.coreAddress, senderAddr, vaa, signatureSet.publicKey));
        postVaaTx.recentBlockhash = blockhash;
        postVaaTx.feePayer = senderAddr;
        yield this.createUnsignedTx(postVaaTx, 'Redeem.PostVAA');
    }
    createUnsignedTx(txReq, description, parallelizable = false) {
        return new connect_sdk_solana_1.SolanaUnsignedTransaction(txReq, this.network, 'Solana', description, parallelizable);
    }
}
exports.SolanaTokenBridge = SolanaTokenBridge;
