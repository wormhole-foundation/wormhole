"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EvmTokenBridge = void 0;
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
const _1 = require(".");
const connect_sdk_evm_1 = require("@wormhole-foundation/connect-sdk-evm");
class EvmTokenBridge {
    constructor(network, chain, provider, contracts) {
        this.network = network;
        this.chain = chain;
        this.provider = provider;
        this.contracts = contracts;
        this.chainId = connect_sdk_evm_1.evmNetworkChainToEvmChainId.get(network, chain);
        const tokenBridgeAddress = this.contracts.tokenBridge;
        if (!tokenBridgeAddress)
            throw new Error(`Wormhole Token Bridge contract for domain ${chain} not found`);
        this.tokenBridgeAddress = tokenBridgeAddress;
        this.tokenBridge = _1.ethers_contracts.Bridge__factory.connect(this.tokenBridgeAddress, provider);
    }
    static async fromProvider(provider, config) {
        const [network, chain] = await connect_sdk_evm_1.EvmPlatform.chainFromRpc(provider);
        return new EvmTokenBridge(network, chain, provider, config[chain].contracts);
    }
    async isWrappedAsset(token) {
        return await this.tokenBridge.isWrappedAsset(new connect_sdk_evm_1.EvmAddress(token).toString());
    }
    async getOriginalAsset(token) {
        if (!(await this.isWrappedAsset(token)))
            throw (0, connect_sdk_1.ErrNotWrapped)(token.toString());
        const tokenContract = connect_sdk_evm_1.EvmPlatform.getTokenImplementation(this.provider, token.toString());
        const [chain, address] = await Promise.all([
            tokenContract.chainId().then(Number).then(connect_sdk_1.toChainId).then(connect_sdk_1.chainIdToChain),
            tokenContract.nativeContract().then((addr) => new connect_sdk_1.UniversalAddress(addr)),
        ]);
        return { chain, address };
    }
    async hasWrappedAsset(token) {
        try {
            await this.getWrappedAsset(token);
            return true;
        }
        catch (e) { }
        return false;
    }
    async getWrappedAsset(token) {
        const wrappedAddress = await this.tokenBridge.wrappedAsset((0, connect_sdk_1.toChainId)(token.chain), token.address.toUniversalAddress().toString());
        if (wrappedAddress === connect_sdk_evm_1.EvmZeroAddress)
            throw (0, connect_sdk_1.ErrNotWrapped)(token.address.toUniversalAddress().toString());
        return (0, connect_sdk_1.toNative)('Evm', wrappedAddress);
    }
    async isTransferCompleted(vaa) {
        //The double keccak here is neccessary due to a fuckup in the original implementation of the
        //  EVM core bridge:
        //Guardians don't sign messages (bodies) but explicitly hash them via keccak256 first.
        //However, they use an ECDSA scheme for signing where the first step is to hash the "message"
        //  (which at this point is already the digest of the original message/body!)
        //Now, on EVM, ecrecover expects the final digest (i.e. a bytes32 rather than a dynamic bytes)
        //  i.e. it does no hashing itself. Therefore the EVM core bridge has to hash the body twice
        //  before calling ecrecover. But in the process of doing so, it erroneously sets the doubly
        //  hashed value as vm.hash instead of using the only once hashed value.
        //And finally this double digest is then used in a mapping to store whether a VAA has already
        //  been redeemed or not, which is ultimately the reason why we have to keccak the hash one
        //  more time here.
        return this.tokenBridge.isTransferCompleted((0, connect_sdk_1.keccak256)(vaa.hash));
    }
    async *createAttestation(token) {
        const ignoredNonce = 0;
        yield this.createUnsignedTx(await this.tokenBridge.attestToken.populateTransaction(new connect_sdk_evm_1.EvmAddress(token).toString(), ignoredNonce), 'TokenBridge.createAttestation');
    }
    async *submitAttestation(vaa) {
        const func = (await this.hasWrappedAsset({
            ...vaa.payload.token,
        }))
            ? 'updateWrapped'
            : 'createWrapped';
        yield this.createUnsignedTx(await this.tokenBridge[func].populateTransaction((0, connect_sdk_1.serialize)(vaa)), 'TokenBridge.' + func);
    }
    async *transfer(sender, recipient, token, amount, payload) {
        const senderAddr = new connect_sdk_evm_1.EvmAddress(sender).toString();
        const recipientChainId = (0, connect_sdk_1.toChainId)(recipient.chain);
        const recipientAddress = recipient.address
            .toUniversalAddress()
            .toUint8Array();
        if (typeof token === 'string' && token === 'native') {
            const txReq = await (payload === undefined
                ? this.tokenBridge.wrapAndTransferETH.populateTransaction(recipientChainId, recipientAddress, connect_sdk_evm_1.unusedArbiterFee, connect_sdk_evm_1.unusedNonce, { value: amount })
                : this.tokenBridge.wrapAndTransferETHWithPayload.populateTransaction(recipientChainId, recipientAddress, connect_sdk_evm_1.unusedNonce, payload, { value: amount }));
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridge.wrapAndTransferETH' +
                (payload === undefined ? '' : 'WithPayload'));
        }
        else {
            //TODO check for ERC-2612 (permit) support on token?
            const tokenAddr = new connect_sdk_evm_1.EvmAddress(token).toString();
            const tokenContract = connect_sdk_evm_1.EvmPlatform.getTokenImplementation(this.provider, tokenAddr);
            const allowance = await tokenContract.allowance(senderAddr, this.tokenBridge.target);
            if (allowance < amount) {
                const txReq = await tokenContract.approve.populateTransaction(this.tokenBridge.target, amount);
                yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'ERC20.approve of TokenBridge');
            }
            const sharedParams = [
                tokenAddr,
                amount,
                recipientChainId,
                recipientAddress,
            ];
            const txReq = await (payload === undefined
                ? this.tokenBridge.transferTokens.populateTransaction(...sharedParams, connect_sdk_evm_1.unusedArbiterFee, connect_sdk_evm_1.unusedNonce)
                : this.tokenBridge.transferTokensWithPayload.populateTransaction(...sharedParams, connect_sdk_evm_1.unusedNonce, payload));
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridge.transferTokens' +
                (payload === undefined ? '' : 'WithPayload'));
        }
    }
    async *redeem(sender, vaa, unwrapNative = true) {
        const senderAddr = new connect_sdk_evm_1.EvmAddress(sender).toString();
        if (vaa.payloadName === 'TransferWithPayload' &&
            vaa.payload.token.chain !== this.chain) {
            const fromAddr = (0, connect_sdk_1.toNative)(this.chain, vaa.payload.from).unwrap();
            if (fromAddr !== senderAddr)
                throw new Error(`VAA.from (${fromAddr}) does not match sender (${senderAddr})`);
        }
        const wrappedNativeAddr = await this.tokenBridge.WETH();
        const tokenAddr = (0, connect_sdk_1.toNative)(this.chain, vaa.payload.token.address).unwrap();
        if (tokenAddr === wrappedNativeAddr && unwrapNative) {
            const txReq = await this.tokenBridge.completeTransferAndUnwrapETH.populateTransaction((0, connect_sdk_1.serialize)(vaa));
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridge.completeTransferAndUnwrapETH');
        }
        else {
            const txReq = await this.tokenBridge.completeTransfer.populateTransaction((0, connect_sdk_1.serialize)(vaa));
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridge.completeTransfer');
        }
    }
    async getWrappedNative() {
        const address = await this.tokenBridge.WETH();
        return (0, connect_sdk_1.toNative)(this.chain, address);
    }
    createUnsignedTx(txReq, description, parallelizable = false) {
        return new connect_sdk_evm_1.EvmUnsignedTransaction((0, connect_sdk_evm_1.addChainId)(txReq, this.chainId), this.network, this.chain, description, parallelizable);
    }
}
exports.EvmTokenBridge = EvmTokenBridge;
